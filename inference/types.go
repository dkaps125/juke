package inference

type Tool struct {
	Name        string
	Description string
	// map of arg name to type
	Arguments map[string]string
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
	Tools        []Tool
}
