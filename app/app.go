package app

import (
	"encoding/json"
	"log"
	"reflect"

	tea "charm.land/bubbletea/v2"
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
	musicSource   music.Source
	llm           inference.Engine
	program       *tea.Program
	inferenceChan chan (bool)
}

func NewApp(config config.Config) *App {
	musicSource := GetMusicSource(config)

	state := musicSource.CurrentState()
	systemPrompt := inference.GetSystemPrompt(state)
	llm := GetLLMEngine(config, systemPrompt)

	app := App{
		musicSource:   musicSource,
		llm:           llm,
		inferenceChan: make(chan (bool)),
	}

	// Sketchy
	app.program = tea.NewProgram(initialModel(app.prompt, app.inferenceChan))
	return &app
}

func (a *App) Start() {
	authChan := a.musicSource.Authenticate()
	<-authChan

	if _, err := a.program.Run(); err != nil {
		log.Fatal(err)
	}
}

func (a *App) prompt(prompt string) {
	toolCalls := make([]inference.ToolCall, 0)

	for message := range a.llm.PromptLLM(prompt) {
		select {
		case <-a.inferenceChan:
			a.program.Send(completedMessage(""))
			return
		default:
			if len(message.Thinking) > 0 {
				a.program.Send(thinkingMessage(message.Thinking))
			}
			if len(message.Content) > 0 {
				a.program.Send(talkingMessage(message.Content))
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
			toolCallResults[i] = a.callTool(tool)
		}

		// Reset tool calls
		toolCalls = make([]inference.ToolCall, 0)

		for message := range a.llm.ProcessTools(toolCallResults) {
			select {
			case <-a.inferenceChan:
				a.program.Send(completedMessage(""))
				return
			default:
				if len(message.Thinking) > 0 {
					a.program.Send(thinkingMessage(message.Thinking))
				}
				if len(message.Content) > 0 {
					a.program.Send(talkingMessage(message.Content))
				}

				if len(message.ToolCalls) > 0 {
					toolCalls = append(toolCalls, message.ToolCalls...)
				}
			}
		}
	}

	a.program.Send(completedMessage(""))
}

func (a *App) callTool(toolCall inference.ToolCall) inference.ToolResult {
	toolMethod := reflect.ValueOf(a.musicSource).MethodByName(toolCall.Name)
	argPtr := reflect.New(toolMethod.Type().In(0))
	mapToStruct(argPtr.Interface(), toolCall.Arguments)

	arg := reflect.ValueOf(argPtr.Elem().Interface())
	result := toolMethod.Call([]reflect.Value{arg})[0].Interface()
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
