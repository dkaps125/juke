package inference

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"iter"
	"log"
	"reflect"

	"github.com/dkaps125/juke/music"
	"github.com/dkaps125/juke/utils"
	"github.com/ollama/ollama/api"
)

// OllamaEngine is the main struct class for inference via Ollama-hosted models
type OllamaEngine struct {
	client    *api.Client
	messages  []api.Message
	modelName string
	tools     api.Tools
}

var (
	stream          = true // TODO: make this clean to use
	ollamaFormat, _ = json.Marshal(map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"Reason": map[string]string{
					"type": "string",
				},
				"Title": map[string]string{
					"type": "string",
				},
				"Artist": map[string]string{
					"type": "string",
				},
			},
			"required": []string{"Title", "Artist"},
		},
	})
)

// NewOllamaEngine creates a new inference engine backed by Ollama
func NewOllamaEngine(opts ProviderOptions) *OllamaEngine {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	return &OllamaEngine{
		client: client,
		messages: []api.Message{
			{
				Role:    "system",
				Content: opts.SystemPrompt,
			},
		},
		modelName: opts.ModelName,
		tools:     ToFunctions[music.ToolSource]("./music/source.go"),
	}
}

func (e *OllamaEngine) ProcessTools(toolResults []ToolResult) iter.Seq[Message] {
	e.messages = append(e.messages, utils.Map(toolResults, func(result ToolResult) api.Message {
		return api.Message{
			Role:       "tool",
			Content:    result.Content,
			ToolCallID: result.ID,
		}
	})...)

	return e.runInference()
}

// PromptLLM does what it says
func (e *OllamaEngine) PromptLLM(userPrompt string) iter.Seq[Message] {
	prompt := getPrompt(userPrompt)

	e.messages = append(e.messages, api.Message{
		Role:    "user",
		Content: prompt,
	})

	return e.runInference()
}

func (e *OllamaEngine) runInference() iter.Seq[Message] {
	ctx := context.Background()

	req := &api.ChatRequest{
		Model:    e.modelName,
		Messages: e.messages,
		Stream:   &stream,
		Options:  map[string]any{"temperature": 0.2, "top_p": 0.9},
		Tools:    e.tools,

		// Use structured outputs to support models without tool calling
		// Format: ollamaFormat,
	}

	messageChan := make(chan (Message))
	allContent := ""
	allThinking := ""
	allToolCalls := make([]api.ToolCall, 0)

	respFunc := func(resp api.ChatResponse) error {
		allContent = allContent + resp.Message.Content
		allThinking = allThinking + resp.Message.Thinking
		allToolCalls = append(allToolCalls, resp.Message.ToolCalls...)

		message := Message{
			Thinking: resp.Message.Thinking,
			Content:  resp.Message.Content,
			ToolCalls: utils.Map(resp.Message.ToolCalls, func(toolCall api.ToolCall) ToolCall {
				return ToolCall{
					ID:        toolCall.ID,
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments.ToMap(),
				}
			}),
		}

		messageChan <- message

		if resp.Done {
			close(messageChan)
			e.messages = append(e.messages, api.Message{
				Role:      "assistant",
				Content:   allContent,
				ToolCalls: allToolCalls,
				Thinking:  allThinking,
			})
		}

		return nil
	}

	go e.client.Chat(ctx, req, respFunc)

	return func(yield func(Message) bool) {
		for message := range messageChan {
			if !yield(message) {
				return
			}
		}
	}
}

func getFunctionDocstrings(name, file string) map[string]string {
	docStrings := make(map[string]string)

	fset := token.NewFileSet()
	pkg, _ := parser.ParseFile(fset, file, nil, parser.ParseComments)
	ast.Inspect(pkg, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || name != ts.Name.Name {
			return true
		}

		if it, ok := ts.Type.(*ast.InterfaceType); ok {
			for _, method := range it.Methods.List {
				docStrings[method.Names[0].Name] = method.Doc.Text()
			}
		}
		return true
	})

	return docStrings
}

func ToFunctions[T any](file string) api.Tools {
	v := reflect.TypeFor[T]()
	numTools := v.NumMethod()
	tools := make(api.Tools, numTools)
	docStrings := getFunctionDocstrings(v.Name(), file)

	for i := range numTools {
		method := v.Method(i)

		argStruct := method.Type.In(0) // Relying on struct args
		numArgs := argStruct.NumField()
		args := api.NewToolPropertiesMap()

		for j := range numArgs {
			field := argStruct.Field(j)
			args.Set(field.Name, api.ToolProperty{
				Type: api.PropertyType{field.Type.Name()},
			})
		}

		tools[i] = api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        method.Name,
				Description: docStrings[method.Name],
				Parameters: api.ToolFunctionParameters{
					Type:       "object",
					Properties: args,
				},
			},
		}
	}

	return tools
}
