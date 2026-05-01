package music

// Song represents song
type Song struct {
	Title  string
	Artist string
}

type SongURI struct {
	URI string
}

type SongID struct {
	ID string
}

type PlayerState struct {
	CurrentSong *Song
	Queue       []Song
	Playing     bool
}

type GenericResult struct {
	Status string
}

type Result struct {
	Title   string
	Artists string
	URI     string
}

// ToolSource provides the interface for types that can be called as LLM tools
type ToolSource interface {
	// Searches for a song
	Search(Song) []Result
	// Plays a song
	PlaySong(SongURI) GenericResult
	// Queues a song
	QueueSong(SongID) GenericResult
}

// Source is a source for streaming music and music data
type Source interface {
	ToolSource
	Authenticate() chan (bool)
	Previous()
	Pause()
	Play()
	Next()
	CurrentState() PlayerState
}
