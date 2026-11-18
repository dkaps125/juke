package app

import (
	"log"

	tea "charm.land/bubbletea/v2"
	"github.com/dkaps125/juke/config"
	"github.com/dkaps125/juke/harness"
	"github.com/dkaps125/juke/music"
)

type App struct {
	musicSource music.Source
	harness     *harness.Harness
	program     *tea.Program
	messageChan chan (harness.Message) // TODO: message type with discriminator
}

func NewApp(config config.Config) *App {
	musicSource := getMusicSource(config)

	inferenceChan := make(chan (bool))
	messageChan := make(chan (harness.Message))
	h := harness.NewHarness(musicSource, config, inferenceChan, messageChan)

	app := App{
		musicSource: musicSource,
		harness:     h,
		messageChan: messageChan,
	}

	// Sketchy
	app.program = tea.NewProgram(initialModel(app.harness.Prompt, inferenceChan, h.GetSlashCommandKeys()))
	return &app
}

func getMusicSource(conf config.Config) music.Source {
	switch conf.MusicSource {
	case config.SPOTIFY:
		return music.NewSpotify()
	default:
		return music.NewSpotify()
	}
}

func (a *App) Start() {
	authChan := a.musicSource.Authenticate()
	<-authChan

	go a.messageHandler()

	if _, err := a.program.Run(); err != nil {
		log.Fatal(err)
	}
}

func (a *App) messageHandler() {
	for {
		a.program.Send(<-a.messageChan)
	}
}
