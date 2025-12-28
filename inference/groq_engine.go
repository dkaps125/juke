package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"github.com/conneroisu/groq-go"
	"github.com/conneroisu/groq-go/pkg/schema"
	"github.com/dkaps125/juke/music"
)

var (
	groqDefaultMessages = []groq.ChatCompletionMessage{
		{
			Role:    groq.RoleSystem,
			Content: SYSTEM_PROMPT,
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
	prompt := getPrompt(userPrompt, currentSong)

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
