package inference

type SongOutput struct {
	Artist string
	Title  string
	Reason string
}

type ProviderOptions struct {
	ModelName    string
	SystemPrompt string
}

var (
	outputType []SongOutput
)
