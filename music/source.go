package music

// Song represents song
type Song struct {
	Title  string
	Artist string
}

type PlayerState struct {
	CurrentSong *Song
	Playing     bool
}

// Source is a source for streaming music and music data
type Source interface {
	Authenticate()
	SearchAndPlaySong(song Song)
	Previous()
	Pause()
	Play()
	Next()
	CurrentState() PlayerState
}
