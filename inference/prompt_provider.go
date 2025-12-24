package inference

import (
	"fmt"
	"strings"

	"github.com/dkaps125/juke/music"
)

func getPrompt(userPrompt string, currentSong *music.Song) string {
	if currentSong == nil {
		return strings.TrimSpace(fmt.Sprintf(`
	I'm listening to music. Here is my request for my next songs: %s. Provide a reason why you're suggesting these songs.
	Suggest songs in this order:
1. Songs specifically requested by the user. In this case, ignore the currently playing song.
2. Songs different from what the user is currently listening to, taking previously heard tracks and user sentiment into account.
		`, userPrompt))
	} else {
		return strings.TrimSpace(fmt.Sprintf(`
	I'm listening to music. My current song is %s by %s. Here is my request for my next songs: %s. Provide a reason why you're suggesting these songs.
	Suggest songs in this order:
1. Songs specifically requested by the user. In this case, ignore the currently playing song.
2. Songs different from what the user is currently listening to, taking previously heard tracks and user sentiment into account.
		`, currentSong.Title, currentSong.Artist, userPrompt))
	}
}
