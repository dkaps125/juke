// This example demonstrates how to authenticate with Spotify.
// In order to run this example yourself, you'll need to:
//
//  1. Register an application at: https://developer.spotify.com/my-applications/
//     - Use "http://localhost:8080/callback" as the redirect URI
//  2. Set the SPOTIFY_ID environment variable to the client ID you got in step 1.
//  3. Set the SPOTIFY_SECRET environment variable to the client secret from step 1.
package main

import (
	"runtime"

	"github.com/dkaps125/juke/config"
	"github.com/dkaps125/juke/inference"
	"github.com/dkaps125/juke/music"
	"github.com/joho/godotenv"
)

func init() {
	runtime.LockOSThread()
	godotenv.Load()
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
	default:
		return inference.NewOllamaEngine(inference.OllamaOptions{
			ModelName: conf.ModelName,
		})
	}
}

func main() {
	config := config.GetConfig()

	// TODO: genericize
	spotify := GetMusicSource(config)
	spotify.Authenticate()

	llm := GetLLMEngine(config)

	app := App{
		music: spotify,
		llm:   llm,
	}

	app.InitializeMenubar()
	app.Run()
}
