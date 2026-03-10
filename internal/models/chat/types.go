package chat

import "github.com/nikaydo/personal-assistant/internal/models/tool"

type Message struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	Refusal    string          `json:"refusal,omitempty"`
	Reasoning  string          `json:"reasoning,omitempty"`
	ToolCalls  []tool.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Type       string          `json:"type,omitempty"`
	ID         string          `json:"id,omitempty"`
	CallId     string          `json:"call_id,omitempty"`
	Name       string          `json:"name,omitempty"`
	Arguments  string          `json:"arguments,omitempty"`
	Output     string          `json:"output,omitempty"`
}
