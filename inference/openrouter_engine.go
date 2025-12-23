package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/dkaps125/juke/music"
	"github.com/revrost/go-openrouter"
	"github.com/revrost/go-openrouter/jsonschema"
)

type SongOutput struct {
	Artist string
	Title  string
	Reason string
}

var (
	openrouterDefaultMessages = []openrouter.ChatCompletionMessage{
		openrouter.SystemMessage(strings.TrimSpace(`
You are a music expert. Your job is to return formatted song titles and artists, incorporating previously played tracks and user sentiment in your suggestions.

Be as succinct as possible, and prioritize tool use over text. Return as many songs as are applicable.
		`)),
	}
	output              []SongOutput
	openrouterFormat, _ = jsonschema.GenerateSchemaForType(output)
)

type OpenrouterOptions struct {
	ModelName string
}

type OpenrouterEngine struct {
	client    *openrouter.Client
	messages  []openrouter.ChatCompletionMessage
	modelName string
}

func NewOpenrouterEngine(opts OpenrouterOptions) OpenrouterEngine {
	client := openrouter.NewClient(
		os.Getenv("OPENROUTER_API_KEY"),
	)

	fmt.Println(openrouterFormat)

	return OpenrouterEngine{
		client:    client,
		modelName: opts.ModelName,
		messages:  slices.Clone(openrouterDefaultMessages),
	}
}

func (e OpenrouterEngine) PromptLLM(userPrompt string, currentSong *music.Song, callback func(song []music.Song)) {
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

	e.messages = append(e.messages, openrouter.UserMessage(prompt))
	resp, err := e.client.CreateChatCompletion(
		context.Background(),
		openrouter.ChatCompletionRequest{
			Model:       e.modelName,
			Temperature: 0.2,
			TopP:        0.9,
			Messages:    e.messages,
			ResponseFormat: &openrouter.ChatCompletionResponseFormat{
				Type: openrouter.ChatCompletionResponseFormatTypeJSONSchema,
				JSONSchema: &openrouter.ChatCompletionResponseFormatJSONSchema{
					Name:   "songs",
					Schema: openrouterFormat,
					Strict: true,
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return
	}

	content := resp.Choices[0].Message.Content.Text
	fmt.Printf("Returned songs: %s\n", content)
	e.messages = append(e.messages, openrouter.AssistantMessage(content))

	var songs []music.Song
	json.Unmarshal([]byte(content), &songs)

	callback(songs)
}
