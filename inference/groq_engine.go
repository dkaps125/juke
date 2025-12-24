package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/conneroisu/groq-go"
	"github.com/conneroisu/groq-go/pkg/schema"
	"github.com/dkaps125/juke/music"
)

var (
	groqDefaultMessages = []groq.ChatCompletionMessage{
		{
			Role: groq.RoleSystem,
			Content: strings.TrimSpace(`
You are a music expert. Your job is to return formatted song titles and artists, incorporating previously played tracks and user sentiment in your suggestions.

Be as succinct as possible, and prioritize tool use over text. Return as many songs as are applicable.
		`),
		},
	}
	groqFormat, _ = schema.ReflectSchema(outputType)
)

type GroqOptions struct {
	ModelName string
}

type GroqEngine struct {
	client    *groq.Client
	messages  []groq.ChatCompletionMessage
	modelName string
}

func NewGroqEngine(opts GroqOptions) GroqEngine {
	client, _ := groq.NewClient(
		os.Getenv("GROQ_API_KEY"),
	)

	return GroqEngine{
		client:    client,
		modelName: opts.ModelName,
		messages:  slices.Clone(groqDefaultMessages),
	}
}

func (e GroqEngine) PromptLLM(userPrompt string, currentSong *music.Song, callback func(song []music.Song)) {
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

	e.messages = append(e.messages, groq.ChatCompletionMessage{
		Content: prompt,
		Role:    groq.RoleUser,
	})
	resp, err := e.client.ChatCompletion(
		context.Background(),
		groq.ChatCompletionRequest{
			Model:       groq.ChatModel(e.modelName),
			Temperature: 0.2,
			TopP:        0.9,
			Messages:    e.messages,
			ResponseFormat: &groq.ChatResponseFormat{
				Type: groq.FormatJSONSchema,
				JSONSchema: &groq.JSONSchema{
					Name:   "songs",
					Schema: *groqFormat,
					Strict: false,
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return
	}

	content := resp.Choices[0].Message.Content
	fmt.Printf("Returned songs: %s\n", content)
	e.messages = append(e.messages, groq.ChatCompletionMessage{
		Content: content,
		Role:    groq.RoleAssistant,
	})

	var songs []music.Song
	json.Unmarshal([]byte(content), &songs)

	callback(songs)
}
