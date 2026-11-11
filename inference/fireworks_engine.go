package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log"
	"reflect"

	"github.com/ZaguanLabs/fireworks-sdk-go"
	"github.com/ZaguanLabs/fireworks-sdk-go/types"
	"github.com/dkaps125/juke/music"
	"github.com/dkaps125/juke/utils"
)

type FireworksEngine struct {
	client    *fireworks.Client
	messages  []types.SharedParamsChatMessage
	modelName string
	tools     []types.SharedParamsChatCompletionTool
}

type PendingToolCall struct {
	name      string
	arguments string
	id        string
}

func NewFireworksEngine(apiKey string, opts ProviderOptions) *FireworksEngine {
	client, err := fireworks.NewClient(
		fireworks.WithAPIKey(apiKey),
	)

	if err != nil {
		log.Fatal(err)
	}

	return &FireworksEngine{
		client: client,
		messages: []types.SharedParamsChatMessage{
			{
				Role:    "system",
				Content: opts.SystemPrompt,
			},
		},
		modelName: opts.ModelName,
		tools:     toFireworksTools[music.ToolSource]("./music/source.go"),
	}
}

func (e *FireworksEngine) ProcessTools(toolResults []ToolResult) iter.Seq[Message] {
	e.messages = append(e.messages, utils.Map(toolResults, func(result ToolResult) types.SharedParamsChatMessage {
		return types.SharedParamsChatMessage{
			Role:       "tool",
			Content:    result.Content,
			ToolCallID: &result.ID,
		}
	})...)

	return e.runInference()
}

func (e *FireworksEngine) PromptLLM(userPrompt string) iter.Seq[Message] {
	prompt := getPrompt(userPrompt)

	e.messages = append(e.messages, types.SharedParamsChatMessage{
		Role:    "user",
		Content: prompt,
	})

	return e.runInference()
}

func (e *FireworksEngine) runInference() iter.Seq[Message] {
	ctx := context.Background()

	stream, err := e.client.Chat.Completions.CreateTypedStream(ctx, types.ChatCompletionCreateParams{
		Model:    e.modelName,
		Messages: e.messages,
		Tools:    e.tools,
	})

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return func(yield func(Message) bool) {
			return
		}
	}

	allContent := ""
	allThinking := ""
	allToolCalls := make([]PendingToolCall, 0)
	var pendingToolCall *PendingToolCall = nil

	return func(yield func(Message) bool) {
		defer stream.Close()

		for stream.Next() {
			completedToolCalls := make([]PendingToolCall, 0)

			chunk := stream.Current()
			if len(chunk.Choices) > 0 {
				choice := chunk.Choices[0]
				content := getOrDefault(choice.Delta.Content, "")
				thinking := getOrDefault(choice.Delta.ReasoningContent, "")

				allContent = allContent + content
				allThinking = allThinking + thinking

				for _, respToolCall := range choice.Delta.ToolCalls {
					name := ""
					arguments := ""

					function := respToolCall.Function.(map[string]interface{})
					if fName, ok := function["name"]; ok {
						name = fName.(string)
					}
					if fArgs, ok := function["arguments"]; ok {
						arguments = fArgs.(string)
					}

					if pendingToolCall != nil {
						pendingToolCall.name += name
						pendingToolCall.arguments += arguments
					} else {
						pendingToolCall = &PendingToolCall{
							id:        *respToolCall.ID,
							name:      name,
							arguments: arguments,
						}
					}

					// Try parsing the args for the pending tool call -- if they parse, mark it complete
					err := json.Unmarshal([]byte(pendingToolCall.arguments), &map[string]any{})
					if err == nil {
						allToolCalls = append(allToolCalls, *pendingToolCall)
						completedToolCalls = append(completedToolCalls, *pendingToolCall)
						pendingToolCall = nil
					}
				}

				message := Message{
					Thinking: thinking,
					Content:  content,
					ToolCalls: utils.Map(completedToolCalls, func(toolCall PendingToolCall) ToolCall {
						var args map[string]any
						json.Unmarshal([]byte(toolCall.arguments), &args)
						return ToolCall{
							ID:        toolCall.id,
							Name:      toolCall.name,
							Arguments: args,
						}
					}),
				}

				if !yield(message) {
					return
				}
			}
		}

		e.messages = append(e.messages, types.SharedParamsChatMessage{
			Role:             "assistant",
			Content:          allContent,
			ReasoningContent: &allThinking,
			ToolCalls: utils.Map(allToolCalls, func(toolCall PendingToolCall) types.SharedParamsChatCompletionMessageToolCall {
				return types.SharedParamsChatCompletionMessageToolCall{
					ID:   &toolCall.id,
					Type: "function",
					Function: types.SharedChatCompletionMessageToolCallFunction{
						Name:      &toolCall.name,
						Arguments: toolCall.arguments,
					},
				}
			}),
		})
	}
}

func getOrDefault(v *string, fallback string) string {
	if v == nil {
		return fallback
	}
	return *v
}

func toFireworksTools[T any](file string) []types.SharedParamsChatCompletionTool {
	v := reflect.TypeFor[T]()
	numTools := v.NumMethod()
	toolList := make([]types.SharedParamsChatCompletionTool, numTools)
	docStrings := getFunctionDocstrings(v.Name(), file)

	for i := range numTools {
		method := v.Method(i)
		args := make(map[string]any)

		if method.Type.NumIn() > 0 {
			argStruct := method.Type.In(0) // Relying on struct args
			numArgs := argStruct.NumField()

			for j := range numArgs {
				field := argStruct.Field(j)
				args[field.Name] = map[string]any{
					"type": field.Type.Name(),
				}
			}
		}

		description := docStrings[method.Name]
		toolList[i] = types.SharedParamsChatCompletionTool{
			Type: "function",
			Function: &types.SharedParamsChatCompletionToolFunction{
				Name:        method.Name,
				Description: &description,
				Parameters: map[string]any{
					"type":       "object",
					"properties": args,
				},
			},
		}
	}

	return toolList
}
