package inference

import (
	"context"
	"fmt"
	"iter"
	"os"

	"github.com/conneroisu/groq-go"
	"github.com/conneroisu/groq-go/pkg/tools"
	"github.com/dkaps125/juke/utils"
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
		tools: toGroqTools(opts.Tools),
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

func toGroqTools(rawTools []Tool) []tools.Tool {
	toolList := make([]tools.Tool, len(rawTools))

	for i, tool := range rawTools {

		args := make(map[string]tools.PropertyDefinition)
		if len(tool.Arguments) > 0 {
			for argName, argType := range tool.Arguments {
				args[argName] = tools.PropertyDefinition{
					Type: argType,
				}
			}
		}

		toolList[i] = tools.Tool{
			Type: "function",
			Function: tools.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
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
