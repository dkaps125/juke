package main

import (
	"fmt"
	"runtime"
	"time"

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

func (a App) InitializeMenubar() {

	menuet.App().Children = func() []menuet.MenuItem {
		return []menuet.MenuItem{
			{
				Text:  menuet.Defaults().String("nowPlayingSong"),
				State: false,
			},
			{
				Text:  menuet.Defaults().String("nowPlayingArtist"),
				State: false,
			},
			{
				Type: menuet.Separator,
			},
			{
				Text: "Previous",
				Clicked: func() {
					a.music.Previous()
				},
			},
			{
				Text: menuet.Defaults().String("playPause"),
				Clicked: func() {
					curr := a.music.CurrentState()

					if curr.Playing {
						a.music.Pause()
						menuet.Defaults().SetString("playPause", "Play")
					} else {
						a.music.Play()
						menuet.Defaults().SetString("playPause", "Pause")
					}

				},
			},
			{
				Text: "Next",
				Clicked: func() {
					a.music.Next()
				},
			},
			{
				Type: menuet.Separator,
			},
			{
				Text: "Juke",
				Clicked: func() {
					response := menuet.App().Alert(menuet.Alert{
						MessageText: "What would you like to hear next?",
						Inputs:      []string{"Prompt"},
						Buttons:     []string{"Juke", "Cancel"},
					})

					if response.Button == 0 && len(response.Inputs) == 1 && response.Inputs[0] != "" {
						prompt := response.Inputs[0]
						a.llm.PromptLLM(prompt, &music.Song{
							Title:  menuet.Defaults().String("nowPlayingSong"),
							Artist: menuet.Defaults().String("nowPlayingArtist"),
						}, func(songs []music.Song) {
							a.music.SearchAndPlaySongs(songs)
						})
					}
				},
			},
		}
	}

	menuet.App().Label = "com.github.dkaps125.juke"
	menuet.App().SetMenuState(&menuet.MenuState{
		Title: "Now Playing",
	})
}

func (a App) startUpdates() {
	go func() {
		for {
			if a.music == nil {
				continue
			}

			curr := a.music.CurrentState()

			if curr.Playing {
				menuet.Defaults().SetString("playPause", "Pause")
			} else {
				menuet.Defaults().SetString("playPause", "Play")
			}

			if curr.CurrentSong == nil {
				continue
			}

			menuet.Defaults().SetString("nowPlayingSong", curr.CurrentSong.Title)
			menuet.Defaults().SetString("nowPlayingArtist", curr.CurrentSong.Artist)
			time.Sleep(time.Second)
		}
	}()
}

func (a App) Run() {
	wg, ctx := menuet.App().GracefulShutdownHandles()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		time.Sleep(3 * time.Second)
	}()

	a.startUpdates()
	menuet.App().RunApplication()
}

func GetMusicSource(conf config.Config) music.Source {
	switch conf.MusicSource {
	case config.SPOTIFY:
		return music.NewSpotify()
	default:
		return music.NewSpotify()
	}
}

func GetLLMEngine(conf config.Config) inference.Engine {
	switch conf.LLMProvider {
	case config.OLLAMA:
		return inference.NewOllamaEngine(inference.OllamaOptions{
			ModelName: conf.ModelName,
		})
	case config.OPENROUTER:
		return inference.NewOpenrouterEngine(inference.OpenrouterOptions{
			ModelName: conf.ModelName,
		})
	case config.GROQ:
		return inference.NewGroqEngine(inference.GroqOptions{
			ModelName: conf.ModelName,
		})
	default:
		return inference.NewOllamaEngine(inference.OllamaOptions{
			ModelName: conf.ModelName,
		})
	}
}

func main() {
	config := config.GetConfig()

	fmt.Println(config)

	// TODO: genericize
	musicSource := GetMusicSource(config)
	musicSource.Authenticate()

	llm := GetLLMEngine(config)

	app := App{
		music: musicSource,
		llm:   llm,
	}

	app.InitializeMenubar()
	app.Run()
}
