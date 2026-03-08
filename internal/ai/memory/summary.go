package memory

import (
	"fmt"

	"github.com/nikaydo/personal-assistant/internal/ai/tools"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/models"
)

func (m *Memory) SummaryShortMemory(prompt string, Queue *llmcalls.Queue, model string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	targetCount := m.Cfg.ShortMemoryMessagesCount
	thresholdCount := m.Cfg.ShortMemoryMessagesCount + m.Cfg.SummaryMemoryStep

	if m.Tokens.MessageCount != thresholdCount {
		return nil
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

	respLLM, err := Queue.AddToQueue(llmcalls.QueueItem{Body: models.RequestBody{
		Model:       model,
		Messages:    msg,
		ToolsChoise: "required",
		Tools:       tools.GetToolLongTerm(),
	}})
	if err != nil {
		return err
	}
	fmt.Println("----------------\n", respLLM, "\n----------------")
	m.Logger.Memory("SummaryShortMemory: summarized short-term memory and updated long-term memory", "short_term_count", len(m.ShortTerm), "long_term_count", len(m.LongTerm))
	m.LongTerm = append(m.LongTerm, History{Question: ShotTermQuestion{Text: respLLM.Choices[0].Message.Content}})
	return nil
}
