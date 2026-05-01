package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

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
		fmt.Print("> ")
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
	switch toolCall.Name {
	case "Search":
		result := a.musicSource.Search(music.Song{
			Artist: toolCall.Arguments["Artist"].(string),
			Title:  toolCall.Arguments["Title"].(string),
		})
		resultBytes, _ := json.Marshal(result)

		return inference.ToolResult{
			Name:    toolCall.Name,
			Content: string(resultBytes),
			ID:      toolCall.ID,
		}
	default:
		return inference.ToolResult{
			Name:    toolCall.Name,
			Content: "Tool not found",
			ID:      toolCall.ID,
		}
	}
}
