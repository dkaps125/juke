package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/caseymrm/menuet"
	"github.com/dkaps125/juke/config"
	"github.com/dkaps125/juke/inference"
	"github.com/dkaps125/juke/music"
	"github.com/joho/godotenv"
)

func init() {
	runtime.LockOSThread()
	godotenv.Load()
}

// App is the main app object
type App struct {
	music music.Source
	llm   inference.Engine
}

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
	case config.OLLAMA:
		return inference.NewOllamaEngine(inference.ProviderOptions{
			ModelName:    conf.ModelName,
			SystemPrompt: systemPromopt,
		})
	case config.OPENROUTER:
		return inference.NewOpenrouterEngine(inference.ProviderOptions{
			ModelName:    conf.ModelName,
			SystemPrompt: systemPromopt,
		})
	case config.GROQ:
		return inference.NewGroqEngine(inference.ProviderOptions{
			ModelName:    conf.ModelName,
			SystemPrompt: systemPromopt,
		})
	default:
		return inference.NewOllamaEngine(inference.ProviderOptions{
			ModelName:    conf.ModelName,
			SystemPrompt: systemPromopt,
		})
	}
}

func main() {
	newSession := flag.Bool("new", false, "Set to ignore currently playing and queued songs")

	config := config.GetConfig()

	musicSource := GetMusicSource(config)
	authChan := musicSource.Authenticate()

	<-authChan

	var state music.PlayerState
	if *newSession {
		state = music.PlayerState{}
	} else {
		state = musicSource.CurrentState()
	}
	systemPrompt := inference.GetSystemPrompt(state)
	llm := GetLLMEngine(config, systemPrompt)

	app := App{
		music: musicSource,
		llm:   llm,
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		prompt, _ := reader.ReadString('\n')

		app.llm.PromptLLM(prompt, &music.Song{
			Title:  menuet.Defaults().String("nowPlayingSong"),
			Artist: menuet.Defaults().String("nowPlayingArtist"),
		}, func(songs []music.Song) {
			for _, song := range songs {
				fmt.Printf("%s -- %s\n", song.Title, song.Artist)
			}
			app.music.SearchAndPlaySongs(songs)
		})
	}

}
