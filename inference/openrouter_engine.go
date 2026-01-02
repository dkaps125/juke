package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/dkaps125/juke/music"
	"github.com/revrost/go-openrouter"
	"github.com/revrost/go-openrouter/jsonschema"
)

var (
	openrouterFormat, _ = jsonschema.GenerateSchemaForType(outputType)
)

type OpenrouterEngine struct {
	client    *openrouter.Client
	messages  []openrouter.ChatCompletionMessage
	modelName string
}

func NewOpenrouterEngine(opts ProviderOptions) *OpenrouterEngine {
	client := openrouter.NewClient(
		os.Getenv("OPENROUTER_API_KEY"),
	)

	return &OpenrouterEngine{
		client:    client,
		modelName: opts.ModelName,
		messages: []openrouter.ChatCompletionMessage{
			openrouter.SystemMessage(opts.SystemPrompt),
		},
	}
}

func (e *OpenrouterEngine) PromptLLM(userPrompt string, callback func(song []music.Song)) {
	prompt := getPrompt(userPrompt)

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
