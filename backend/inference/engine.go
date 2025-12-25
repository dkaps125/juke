package inference

import "github.com/dkaps125/juke/music"

// Engine interface describes the action that an LLM engine can take
type Engine interface {
	PromptLLM(userPrompt string, currentSong *music.Song, callback func(song []music.Song))
}
