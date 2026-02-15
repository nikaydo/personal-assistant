package chat

import "github.com/nikaydo/jira-filler/internal/models/tool"

type Message struct {
	Role       string          `json:"role"`
	Content    string          `json:"content,omitempty"`
	Refusal    string          `json:"refusal,omitempty"`
	Reasoning  string          `json:"reasoning,omitempty"`
	ToolCalls  []tool.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}
