package main

import (
	"time"

	"github.com/caseymrm/menuet"
	"github.com/dkaps125/juke/inference"
	"github.com/dkaps125/juke/music"
)

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
						}, func(song music.Song) {
							a.music.SearchAndPlaySong(song)
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
