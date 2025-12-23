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
	ollamaDefaultMessages = []api.Message{
		{
			Role: "system",
			Content: strings.TrimSpace(`
You are a music expert. Your job is to return formatted song titles and artists, incorporating previously played tracks and user sentiment in your suggestions.

Be as succinct as possible, and prioritize tool use over text. Return as many songs as are applicable.
		`),
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
	var prompt string
	if currentSong == nil {
		prompt = strings.TrimSpace(fmt.Sprintf(`
	I'm listening to music. Here is my request for my next songs: %s. Provide a reason why you're suggesting these songs.
	Suggest songs in this order:
1. Songs specifically requested by the user. In this case, ignore the currently playing song.
2. Songs different from what the user is currently listening to, taking previously heard tracks and user sentiment into account.
		`, userPrompt))
	} else {
		prompt = strings.TrimSpace(fmt.Sprintf(`
	I'm listening to music. My current song is %s by %s. Here is my request for my next songs: %s. Provide a reason why you're suggesting these songs.
	Suggest songs in this order:
1. Songs specifically requested by the user. In this case, ignore the currently playing song.
2. Songs different from what the user is currently listening to, taking previously heard tracks and user sentiment into account.
		`, currentSong.Title, currentSong.Artist, userPrompt))
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
