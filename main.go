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

	"github.com/dkaps125/juke/inference"
	"github.com/dkaps125/juke/music"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	// TODO: genericize
	spotify := music.NewSpotify()
	spotify.Authenticate()

	llm := inference.NewOllamaEngine(inference.OllamaOptions{
		ModelName: "gemma3n:e4b",
	})

	app := App{
		music: spotify,
		llm:   llm,
	}

	app.InitializeMenubar()
	app.Run()
}
