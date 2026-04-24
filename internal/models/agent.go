// Package models defines the data structures used by the smart calendar application.
package models

type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

type Message struct {
	Role       MessageRole
	Content    string
	ToolCallID string
	ToolCalls  []ToolCall
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type AgentAction struct {
	FinalAnswer        string
	ClarifyingQuestion string
	ToolCalls          []ToolCall
}

type Chunk struct {
	Text string
	Done bool
}

type StreamEvent struct {
	Chunk  string
	Action *AgentAction
	Err    error
}
