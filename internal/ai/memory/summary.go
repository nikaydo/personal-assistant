package memory

import (
	"github.com/nikaydo/personal-assistant/internal/models"
)

func (m *Memory) SummaryShortMemory(prompt string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	targetCount := m.Cfg.ShortMemoryMessagesCount
	thresholdCount := m.Cfg.ShortMemoryMessagesCount + m.Cfg.SummaryMemoryStep

	if m.Tokens.MessageCount != thresholdCount {
		return
	}

	msg := []models.Message{{Role: "system", Content: prompt}}
	for m.Tokens.MessageCount > targetCount && len(m.ShortTerm) > 0 {
		msg = append(msg,
			models.Message{
				Role:    "user",
				Content: m.ShortTerm[0].Question.Text,
			}, models.Message{
				Role:    "assistant",
				Content: m.ShortTerm[0].Answer.Text,
			})
		m.ShortTerm = m.ShortTerm[1:]
		m.Tokens.MessageCount--
	}
	//суммироваь msg и сохранить в long-term memory
}
