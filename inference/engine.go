package inference

import "iter"

// Engine interface describes the action that an LLM engine can take
type Engine interface {
	PromptLLM(userPrompt string) iter.Seq[Message]
	ProcessTools(toolResults []ToolResult) iter.Seq[Message]
}
