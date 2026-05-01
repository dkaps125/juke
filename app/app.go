package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/dkaps125/juke/config"
	"github.com/dkaps125/juke/inference"
	"github.com/dkaps125/juke/music"
)

func GetMusicSource(conf config.Config) music.Source {
	switch conf.MusicSource {
	case config.SPOTIFY:
		return music.NewSpotify()
	default:
		return music.NewSpotify()
	}
}

func GetLLMEngine(conf config.Config, systemPromopt string) inference.Engine {
	switch conf.LLMProvider {
	// case config.OLLAMA:
	// 	return inference.NewOllamaEngine(inference.ProviderOptions{
	// 		ModelName:    conf.ModelName,
	// 		SystemPrompt: systemPromopt,
	// 	})
	// case config.OPENROUTER:
	// 	return inference.NewOpenrouterEngine(inference.ProviderOptions{
	// 		ModelName:    conf.ModelName,
	// 		SystemPrompt: systemPromopt,
	// 	})
	// case config.GROQ:
	// 	return inference.NewGroqEngine(inference.ProviderOptions{
	// 		ModelName:    conf.ModelName,
	// 		SystemPrompt: systemPromopt,
	// 	})
	default:
		return inference.NewOllamaEngine(inference.ProviderOptions{
			ModelName:    conf.ModelName,
			SystemPrompt: systemPromopt,
		})
	}
}

type App struct {
	musicSource music.Source
	llm         inference.Engine
}

func NewApp(config config.Config) App {
	musicSource := GetMusicSource(config)

	state := musicSource.CurrentState()
	systemPrompt := inference.GetSystemPrompt(state)
	llm := GetLLMEngine(config, systemPrompt)

	return App{
		musicSource: musicSource,
		llm:         llm,
	}
}

func (a App) Start() {
	authChan := a.musicSource.Authenticate()

	<-authChan

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\n> ")
		prompt, _ := reader.ReadString('\n')
		toolCalls := make([]inference.ToolCall, 0)

		for message := range a.llm.PromptLLM(prompt) {
			if len(message.Thinking) > 0 {
				fmt.Print(message.Thinking)
			}
			if len(message.Content) > 0 {
				fmt.Print(message.Content)
			}

			if len(message.ToolCalls) > 0 {
				toolCalls = append(toolCalls, message.ToolCalls...)
			}
		}

		for len(toolCalls) > 0 {
			toolCallResults := make([]inference.ToolResult, len(toolCalls))
			for i, tool := range toolCalls {
				toolCallResults[i] = a.callTool(tool)
			}

			// Reset tool calls
			toolCalls = make([]inference.ToolCall, 0)

			for message := range a.llm.ProcessTools(toolCallResults) {
				if len(message.Thinking) > 0 {
					fmt.Print(message.Thinking)
				}
				if len(message.Content) > 0 {
					fmt.Print(message.Content)
				}

				if len(message.ToolCalls) > 0 {
					toolCalls = append(toolCalls, message.ToolCalls...)
				}
			}
		}
	}
}

func (a App) callTool(toolCall inference.ToolCall) inference.ToolResult {
	toolMethod := reflect.ValueOf(a.musicSource).MethodByName(toolCall.Name)
	argPtr := reflect.New(toolMethod.Type().In(0))
	mapToStruct(argPtr.Interface(), toolCall.Arguments)

	result := toolMethod.Call([]reflect.Value{reflect.ValueOf(argPtr.Elem().Interface())})[0].Interface()
	resultBytes, _ := json.Marshal(result)

	return inference.ToolResult{
		Name:    toolCall.Name,
		Content: string(resultBytes),
		ID:      toolCall.ID,
	}
}

func mapToStruct(t interface{}, m map[string]any) interface{} {
	jsonBytes, _ := json.Marshal(m)
	json.Unmarshal(jsonBytes, t)

	return t
}
