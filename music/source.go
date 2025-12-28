package music

// Song represents song
type Song struct {
	Title  string
	Artist string
}

type PlayerState struct {
	CurrentSong *Song
	Queue       []Song
	Playing     bool
}

// Source is a source for streaming music and music data
type Source interface {
	Authenticate() chan (bool)
	SearchAndPlaySongs(songs []Song)
	Previous()
	Pause()
	Play()
	Next()
	CurrentState() PlayerState
}
