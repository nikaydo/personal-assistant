package ai

import (
	"fmt"

	mod "github.com/nikaydo/personal-assistant/internal/models"
)

func firstChoice(resp mod.ResponseBody) (mod.Message, error) {
	if len(resp.Choices) == 0 {
		return mod.Message{}, fmt.Errorf("empty model response: choices is empty")
	}
	return resp.Choices[0].Message, nil
}
