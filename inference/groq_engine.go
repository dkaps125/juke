package inference

import (
	"context"
	"fmt"
	"iter"
	"os"
	"reflect"

	"github.com/conneroisu/groq-go"
	"github.com/conneroisu/groq-go/pkg/schema"
	"github.com/conneroisu/groq-go/pkg/tools"
	"github.com/dkaps125/juke/music"
	"github.com/dkaps125/juke/utils"
)

var (
	groqFormat, _ = schema.ReflectSchema(outputType)
)

type GroqEngine struct {
	client    *groq.Client
	messages  []groq.ChatCompletionMessage
	modelName string
	tools     []tools.Tool
}

func NewGroqEngine(opts ProviderOptions) *GroqEngine {
	client, _ := groq.NewClient(
		os.Getenv("GROQ_API_KEY"),
	)

	return &GroqEngine{
		client:    client,
		modelName: opts.ModelName,
		messages: []groq.ChatCompletionMessage{
			{
				Role:    groq.RoleSystem,
				Content: opts.SystemPrompt,
			},
		},
		tools: toGroqTools[music.ToolSource]("./music/source.go"),
	}
}

func (e *GroqEngine) ProcessTools(toolResults []ToolResult) iter.Seq[Message] {
	e.messages = append(e.messages, utils.Map(toolResults, func(result ToolResult) groq.ChatCompletionMessage {
		return groq.ChatCompletionMessage{
			Role:       groq.RoleTool,
			Content:    result.Content,
			ToolCallID: result.ID,
		}
	})...)

	return e.runInference()
}

func (e *GroqEngine) PromptLLM(userPrompt string) iter.Seq[Message] {
	prompt := getPrompt(userPrompt)

	e.messages = append(e.messages, groq.ChatCompletionMessage{
		Role:    groq.RoleUser,
		Content: prompt,
	})

	return e.runInference()
}

func (e *GroqEngine) runInference() iter.Seq[Message] {
	stream, err := e.client.ChatCompletionStream(
		context.Background(),
		groq.ChatCompletionRequest{
			Model:       groq.ChatModel(e.modelName),
			Temperature: 0.2,
			TopP:        0.9,
			MaxTokens:   1000,
			Messages:    e.messages,
			Tools:       e.tools,
			Stream:      true,
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return func(yield func(Message) bool) {
			return
		}
	}

	allContent := ""
	allToolCalls := make([]tools.ToolCall, 0)

	return func(yield func(Message) bool) {
		for {
			resp, err := stream.Recv()

			// EOF here indicates end of stream
			if err != nil {
				e.messages = append(e.messages, groq.ChatCompletionMessage{
					Role:      groq.RoleAssistant,
					Content:   allContent,
					ToolCalls: allToolCalls,
				})

				return
			}

			allContent = allContent + resp.Choices[0].Delta.Content
			allToolCalls = append(allToolCalls, resp.Choices[0].Delta.ToolCalls...)
			message := Message{
				Thinking: "",
				Content:  resp.Choices[0].Delta.Content,
				ToolCalls: utils.Map(resp.Choices[0].Delta.ToolCalls, func(toolCall tools.ToolCall) ToolCall {
					return ToolCall{
						ID:   toolCall.ID,
						Name: toolCall.Function.Name,
					}
				}),
			}

			if !yield(message) {
				return
			}
		}
	}
}

func toGroqTools[T any](file string) []tools.Tool {
	v := reflect.TypeFor[T]()
	numTools := v.NumMethod()
	toolList := make([]tools.Tool, numTools)
	docStrings := getFunctionDocstrings(v.Name(), file)

	for i := range numTools {
		method := v.Method(i)

		argStruct := method.Type.In(0) // Relying on struct args
		numArgs := argStruct.NumField()
		args := make(map[string]tools.PropertyDefinition)

		for j := range numArgs {
			field := argStruct.Field(j)
			args[field.Name] = tools.PropertyDefinition{
				Type: field.Type.Name(),
			}
		}

		toolList[i] = tools.Tool{
			Type: "function",
			Function: tools.FunctionDefinition{
				Name:        method.Name,
				Description: docStrings[method.Name],
				Parameters: tools.FunctionParameters{
					Type:       "object",
					Properties: args,
					Required:   []string{}, // TODO
				},
			},
		}
	}

	return toolList
}
