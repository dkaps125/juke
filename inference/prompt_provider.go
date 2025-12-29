package inference

import (
	"fmt"
	"strings"

	"github.com/dkaps125/juke/music"
)

func GetSystemPrompt(state music.PlayerState) string {
	basePrompt := `
# Context
You are a music expert. Your job is to return formatted song titles and artists, incorporating previously played tracks and user sentiment in your suggestions.
Be as succinct as possible, and prioritize tool use over text. Return as many songs as are applicable.

# User recommendations
The user is listening to music. They will give you a request for music to listen to. Provide a reason why you're suggesting each song.

Suggest songs in this order:
1. Songs specifically requested by the user
2. Songs that are similar to what you have already suggested, taking previously heard tracks and user sentiment into account.

# Currently playing song
`

	if state.CurrentSong != nil {
		basePrompt += fmt.Sprintf(`- The user is currently listening to %s, by %s.
`, state.CurrentSong.Title, state.CurrentSong.Artist)
	} else {
		basePrompt += `- The user is not currently listening to anything.
`
	}

	basePrompt += `
# Queued songs
`
	if state.Queue != nil && len(state.Queue) > 0 {
		for _, song := range state.Queue {
			basePrompt += fmt.Sprintf(`- %s by %s
`, song.Title, song.Artist)
		}
	} else {
		basePrompt += `- The user has no songs in their queue.
`
	}

	return strings.TrimSpace(basePrompt)
}

func getPrompt(userPrompt string, currentSong *music.Song) string {
	return userPrompt
}
