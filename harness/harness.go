package harness

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"

	"github.com/dkaps125/juke/config"
	"github.com/dkaps125/juke/inference"
	"github.com/dkaps125/juke/music"
)

type Harness struct {
	music.Source
	llm           inference.Engine
	inferenceChan chan (bool)
	messageChan   chan (Message)

	tasks map[string]Task
}

func NewHarness(musicSource music.Source, config config.Config, inferenceChan chan (bool), messageChan chan (Message)) *Harness {
	systemPrompt := inference.GetSystemPrompt()
	tools := append(toTools[music.ToolSource]("./music/source.go"), toTools[ToolSource]("./harness/source.go")...)
	llm := getLLMEngine(config, systemPrompt, tools)
	tasks := make(map[string]Task)

	return &Harness{
		musicSource,
		llm,
		inferenceChan,
		messageChan,
		tasks,
	}
}

func getLLMEngine(conf config.Config, systemPrompt string, tools []inference.Tool) inference.Engine {
	switch conf.LLMProvider {
	case config.OLLAMA:
		return inference.NewOllamaEngine(inference.ProviderOptions{
			ModelName:    conf.ModelName,
			SystemPrompt: systemPrompt,
		})
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
