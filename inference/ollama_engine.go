package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strings"

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
	defaultMessages = []api.Message{
		{
			Role: "system",
			Content: strings.TrimSpace(`
You are a music expert. Be as succinct as possible, and prioritize tool use over text.
You should suggest specific songs whenever possible, and prioritize songs that are not the same as what the user is currently listening to.
If a specific song is requested, prioritizing playing that song.
		`),
		},
	}
	stream    = false
	format, _ = json.Marshal(map[string]any{
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
		messages:  slices.Clone(defaultMessages),
		modelName: opts.ModelName,
	}
}

// PromptLLM does what it says
func (e OllamaEngine) PromptLLM(userPrompt string, currentSong *music.Song, callback func(song music.Song)) {
	var prompt string
	if currentSong == nil {
		prompt = fmt.Sprintf("I'm listening to music. Play my next song based on the following criteria: %s", userPrompt)
	} else {
		prompt = fmt.Sprintf("I'm listening to music. My current song is %s by %s. Play my next song based on the following criteria: %s. Provide a reason why you're suggesting this song.", currentSong.Title, currentSong.Artist, userPrompt)
	}

	e.messages = append(e.messages, api.Message{
		Role:    "user",
		Content: prompt,
	})

	ctx := context.Background()

	req := &api.ChatRequest{
		Model:    e.modelName,
		Messages: e.messages,
		Stream:   &stream,
		Options:  map[string]any{"temperature": 0.9, "top_p": 0.9},

		// Use structured outputs to support models without tool calling
		Format: format,
	}

	respFunc := func(resp api.ChatResponse) error {
		content := resp.Message.Content
		e.messages = append(e.messages, api.Message{
			Role:    "assistant",
			Content: content,
		})

		var song music.Song
		json.Unmarshal([]byte(content), &song)

		callback(song)
		return nil
	}

	err := e.client.Chat(ctx, req, respFunc)
	if err != nil {
		log.Fatal(err)
	}
}
