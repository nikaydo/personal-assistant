package ai

import (
	"testing"

	chatcommand "github.com/nikaydo/personal-assistant/internal/ai/chatCommand"
)

func TestComposeSystemPrompt_AppendsOnce(t *testing.T) {
	base := "base"
	param := chatcommand.Param{SystemPrompt: " + command"}

	got := composeSystemPrompt(base, param)
	want := "base + command"
	if got != want {
		t.Fatalf("unexpected system prompt: got=%q want=%q", got, want)
	}
}
