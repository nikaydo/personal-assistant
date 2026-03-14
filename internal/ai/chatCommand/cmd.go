package chatcommand

import (
	"strings"

	"github.com/nikaydo/personal-assistant/internal/agent"
	"github.com/nikaydo/personal-assistant/internal/models"
)

type Param struct {
	SystemPrompt string
	Tool         []models.Tool
}

func CheckCmd(str string) (Param, bool) {
	spl := strings.Split(str, " ")
	if len(spl) <= 1 {
		return Param{}, false
	}
	switch spl[0] {
	case CallAgent:
		return Param{
			SystemPrompt: ToolPrompt,
			Tool:         agent.GetToolDefault()}, true
	default:
		return Param{SystemPrompt: ReqNoTool}, false
	}
}
