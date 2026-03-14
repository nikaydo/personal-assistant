package chatcommand

import (
	"strings"

	"github.com/nikaydo/personal-assistant/internal/agent"
	"github.com/nikaydo/personal-assistant/internal/models"
)

type Param struct {
	SystemPrompt string
	Tool         []models.Tool
	Message      string
}

func CheckCmd(str string) (Param, bool) {
	trimmed := strings.TrimSpace(str)
	spl := strings.Fields(trimmed)
	if len(spl) <= 1 {
		return Param{}, false
	}
	message := strings.TrimSpace(strings.TrimPrefix(trimmed, spl[0]))
	switch spl[0] {
	case CallAgent:
		return Param{
			SystemPrompt: ToolPrompt,
			Tool:         agent.GetToolDefault(),
			Message:      message,
		}, true
	case CallWebSearch:
		return Param{
			SystemPrompt: WebSearchPrompt,
			Message:      message,
		}, true
	default:
		return Param{SystemPrompt: ReqNoTool}, false
	}
}
