package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"

	"github.com/dkaps125/juke/music"
	"github.com/ollama/ollama/api"
)

// OllamaOptions contains options that will be used to run inference through Ollama
type OllamaOptions struct {
	ModelName string
}

// OllamaEngine is the main struct class for inference via Ollama-hosted models
type OllamaEngine struct {
	client    *api.Client
	messages  []api.Message
	modelName string
}

var (
	ollamaDefaultMessages = []api.Message{
		{
			Role:    "system",
			Content: SYSTEM_PROMPT,
		},
	}
	stream          = false
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
func NewOllamaEngine(opts OllamaOptions) Engine {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	return OllamaEngine{
		client:    client,
		messages:  slices.Clone(ollamaDefaultMessages),
		modelName: opts.ModelName,
	}
}

// PromptLLM does what it says
func (e OllamaEngine) PromptLLM(userPrompt string, currentSong *music.Song, callback func(song []music.Song)) {
	prompt := getPrompt(userPrompt, currentSong)

	e.messages = append(e.messages, api.Message{
		Role:    "user",
		Content: prompt,
	})

	ctx := context.Background()

	req := &api.ChatRequest{
		Model:    e.modelName,
		Messages: e.messages,
		Stream:   &stream,
		Options:  map[string]any{"temperature": 0.2, "top_p": 0.9},

		// Use structured outputs to support models without tool calling
		Format: ollamaFormat,
	}

	respFunc := func(resp api.ChatResponse) error {
		content := resp.Message.Content
		e.messages = append(e.messages, api.Message{
			Role:    "assistant",
			Content: content,
		})

		fmt.Printf("Returned songs: %s\n", content)

		var songs []music.Song
		json.Unmarshal([]byte(content), &songs)

		callback(songs)
		return nil
	}

	err := e.client.Chat(ctx, req, respFunc)
	if err != nil {
		log.Fatal(err)
	}
}
