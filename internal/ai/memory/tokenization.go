package memory

import (
	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/models"
)

type ContextTokens struct {
	SystemPromptPercent int
	SystemMemoryLimit   int
	UserProfileLimit    int
	ToolsMemoryLimit    int
	LongTermLimit       int
	ShortTermLimit      int

	ContextLimit int

	MessageCount int

	ContextCoeff          []float32
	CountSymbolsInContext int
}

func (ct *ContextTokens) CalculateContextLimit(cfg config.Config) {
	ct.SystemMemoryLimit = int(float32(ct.ContextLimit) * float32(cfg.SystemMemoryPercent) / 100)
	ct.UserProfileLimit = int(float32(ct.ContextLimit) * float32(cfg.UserProfilePercent) / 100)
	ct.ToolsMemoryLimit = int(float32(ct.ContextLimit) * float32(cfg.ToolsMemoryPercent) / 100)
	ct.LongTermLimit = int(float32(ct.ContextLimit) * float32(cfg.LongTermPercent) / 100)
	ct.ShortTermLimit = int(float32(ct.ContextLimit) * float32(cfg.ShortTermPercent) / 100)
	ct.SystemPromptPercent = int(float32(ct.ContextLimit) * float32(cfg.SystemPromptPercent) / 100)
}

func (ct *ContextTokens) GetContextCoeff() float32 {
	var count int
	var totalCoeff float32
	for _, coeff := range ct.ContextCoeff {
		totalCoeff += coeff
		count++
	}

	if count == 0 {
		return 5
	}
	return totalCoeff / float32(count)
}

func (ct *ContextTokens) ContextCoeffCalc(q string, body models.ResponseBody) {
	if len(ct.ContextCoeff) >= ct.ContextLimit {
		ct.ContextCoeff = ct.ContextCoeff[1:]
	}
	ct.ContextCoeff = append(ct.ContextCoeff, float32(ct.CountSymbolsInContext)/float32(body.Usage.TotalTokens))
}
