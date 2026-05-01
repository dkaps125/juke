package inference

type SongOutput struct {
	Artist string
	Title  string
	Reason string
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}

type ToolResult struct {
	Name    string
	Content string
	ID      string
}

type Message struct {
	Thinking  string
	Content   string
	ToolCalls []ToolCall
}

type ProviderOptions struct {
	ModelName    string
	SystemPrompt string
}

var (
	outputType []SongOutput
)
