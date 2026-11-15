package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"

	tea "charm.land/bubbletea/v2"
	"github.com/dkaps125/juke/config"
	"github.com/dkaps125/juke/harness"
	"github.com/dkaps125/juke/inference"
	"github.com/dkaps125/juke/music"
)

type App struct {
	musicSource   music.Source
	harness       *harness.Harness
	program       *tea.Program
	inferenceChan chan (bool)
	messageChan   chan (harness.Message) // TODO: message type with discriminator
}

func GetMusicSource(conf config.Config) music.Source {
	switch conf.MusicSource {
	case config.SPOTIFY:
		return music.NewSpotify()
	default:
		return music.NewSpotify()
	}
}

func GetLLMEngine(conf config.Config, systemPrompt string, tools []inference.Tool) inference.Engine {
	switch conf.LLMProvider {
	// case config.OLLAMA:
	// 	return inference.NewOllamaEngine(inference.ProviderOptions{
	// 		ModelName:    conf.ModelName,
	// 		SystemPrompt: systemPrompt,
	// 	})
	// case config.OPENROUTER:
	// 	return inference.NewOpenrouterEngine(inference.ProviderOptions{
	// 		ModelName:    conf.ModelName,
	// 		SystemPrompt: systemPrompt,
	// 	})
	case config.FIREWORKS:
		return inference.NewFireworksEngine(
			os.Getenv("FIREWORKS_API_KEY"), // TODO: document or config
			inference.ProviderOptions{
				ModelName:    conf.ModelName,
				SystemPrompt: systemPrompt,
				Tools:        tools,
			},
		)
	case config.GROQ:
		return inference.NewGroqEngine(inference.ProviderOptions{
			ModelName:    conf.ModelName,
			SystemPrompt: systemPrompt,
			Tools:        tools,
		})
	default:
		return inference.NewOllamaEngine(inference.ProviderOptions{
			ModelName:    conf.ModelName,
			SystemPrompt: systemPrompt,
			Tools:        tools,
		})
	}
}

func NewApp(config config.Config) *App {
	musicSource := GetMusicSource(config)

	state := musicSource.CurrentState()
	systemPrompt := inference.GetSystemPrompt(state)
	tools := append(toTools[music.ToolSource]("./music/source.go"), toTools[harness.ToolSource]("./harness/source.go")...)
	llm := GetLLMEngine(config, systemPrompt, tools)

	inferenceChan := make(chan (bool))
	messageChan := make(chan (harness.Message))

	app := App{
		musicSource:   musicSource,
		harness:       harness.NewHarness(musicSource, llm, inferenceChan, messageChan),
		inferenceChan: inferenceChan,
		messageChan:   messageChan,
	}

	// Sketchy
	app.program = tea.NewProgram(initialModel(app.harness.Prompt, app.inferenceChan))
	return &app
}

func (a *App) Start() {
	authChan := a.musicSource.Authenticate()
	<-authChan

	go a.MessageHandler()

	if _, err := a.program.Run(); err != nil {
		log.Fatal(err)
	}
}

func (a *App) MessageHandler() {
	for {
		a.program.Send(<-a.messageChan)
	}
}

func toTools[T any](file string) []inference.Tool {
	v := reflect.TypeFor[T]()
	numTools := v.NumMethod()
	toolList := make([]inference.Tool, numTools)
	docStrings := getFunctionDocstrings(v.Name(), file)

	for i := range numTools {
		method := v.Method(i)
		args := make(map[string]string)

		if method.Type.NumIn() > 0 {
			argStruct := method.Type.In(0) // Relying on struct args
			numArgs := argStruct.NumField()

			for j := range numArgs {
				field := argStruct.Field(j)
				args[field.Name] = field.Type.Name()
			}
		}

		description := docStrings[method.Name]
		toolList[i] = inference.Tool{
			Name:        method.Name,
			Description: description,
			Arguments:   args,
		}
	}

	return toolList
}

func getFunctionDocstrings(name, file string) map[string]string {
	docStrings := make(map[string]string)

	fset := token.NewFileSet()
	pkg, _ := parser.ParseFile(fset, file, nil, parser.ParseComments)
	ast.Inspect(pkg, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || name != ts.Name.Name {
			return true
		}

		if it, ok := ts.Type.(*ast.InterfaceType); ok {
			for _, method := range it.Methods.List {
				docStrings[method.Names[0].Name] = method.Doc.Text()
			}
		}
		return true
	})

	return docStrings
}
