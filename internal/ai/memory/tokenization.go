package memory

import (
	"sync"

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

	ContextCoeffCount     int
	ContextCoeff          []float32
	CountSymbolsInContext int

	mu sync.RWMutex
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
	ct.mu.RLock()
	defer ct.mu.RUnlock()

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

func (ct *ContextTokens) ContextCoeffSnapshot() []float32 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	out := make([]float32, len(ct.ContextCoeff))
	copy(out, ct.ContextCoeff)
	return out
}

func (ct *ContextTokens) ContextCoeffCalc(symbolsInContext int, body models.ResponseBody) {
	if body.Usage.TotalTokens <= 0 || symbolsInContext <= 0 {
		return
	}

	window := ct.ContextCoeffCount
	if window <= 0 {
		window = 12
	}

	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.ContextCoeff = append(ct.ContextCoeff, float32(symbolsInContext)/float32(body.Usage.TotalTokens))
	if len(ct.ContextCoeff) > window {
		ct.ContextCoeff = ct.ContextCoeff[len(ct.ContextCoeff)-window:]
	}
}
