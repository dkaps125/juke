package harness

import (
	"encoding/json"
	"reflect"

	"github.com/dkaps125/juke/inference"
	"github.com/dkaps125/juke/music"
)

type Harness struct {
	music.Source
	llm           inference.Engine
	inferenceChan chan (bool)
	messageChan   chan (Message)
}

func NewHarness(musicSource music.Source, llm inference.Engine, inferenceChan chan (bool), messageChan chan (Message)) *Harness {
	return &Harness{
		musicSource,
		llm,
		inferenceChan,
		messageChan,
	}
}

func (h *Harness) Prompt(prompt string) {
	toolCalls := make([]inference.ToolCall, 0)

	for message := range h.llm.PromptLLM(prompt) {
		select {
		case <-h.inferenceChan:
			h.messageChan <- CompletedMessage("")
			return
		default:
			if len(message.Thinking) > 0 {
				h.messageChan <- ThinkingMessage(message.Thinking)
			}
			if len(message.Content) > 0 {
				h.messageChan <- TalkingMessage(message.Content)
			}

			if len(message.ToolCalls) > 0 {
				toolCalls = append(toolCalls, message.ToolCalls...)
			}
		}
	}

	// TODO: handle tool call interruption
	for len(toolCalls) > 0 {
		toolCallResults := make([]inference.ToolResult, len(toolCalls))
		for i, tool := range toolCalls {
			toolCallResults[i] = h.callTool(tool)
		}

		// Reset tool calls
		toolCalls = make([]inference.ToolCall, 0)

		for message := range h.llm.ProcessTools(toolCallResults) {
			select {
			case <-h.inferenceChan:
				h.messageChan <- CompletedMessage("")
				return
			default:
				if len(message.Thinking) > 0 {
					h.messageChan <- ThinkingMessage(message.Thinking)
				}
				if len(message.Content) > 0 {
					h.messageChan <- TalkingMessage(message.Content)
				}

				if len(message.ToolCalls) > 0 {
					toolCalls = append(toolCalls, message.ToolCalls...)
				}
			}
		}
	}

	h.messageChan <- CompletedMessage("")
}

func (h *Harness) callTool(toolCall inference.ToolCall) inference.ToolResult {
	toolMethod := reflect.ValueOf(h).MethodByName(toolCall.Name)
	var result any

	if toolMethod.Type().NumIn() > 0 {
		argPtr := reflect.New(toolMethod.Type().In(0))
		mapToStruct(argPtr.Interface(), toolCall.Arguments)

		arg := reflect.ValueOf(argPtr.Elem().Interface())
		result = toolMethod.Call([]reflect.Value{arg})[0].Interface()
	} else {
		result = toolMethod.Call([]reflect.Value{})[0].Interface()
	}

	resultBytes, _ := json.Marshal(result)

	return inference.ToolResult{
		Name:    toolCall.Name,
		Content: string(resultBytes),
		ID:      toolCall.ID,
	}
}

func mapToStruct(t interface{}, m map[string]any) {
	jsonBytes, _ := json.Marshal(m)
	json.Unmarshal(jsonBytes, t)
}
