package memory

import (
	"strings"

	"github.com/nikaydo/personal-assistant/internal/ai/tools"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/models"
)

var enqueueSummaryFn = func(q *llmcalls.Queue, item llmcalls.QueueItem) (models.ResponseBody, error) {
	return q.AddToQueue(item)
}

var detectChosenToolFn = func(t *tools.Tool, body models.ResponseBody) error {
	return t.DetectChosenTool(body)
}

func (m *Memory) SummaryShortMemory(Queue *llmcalls.Queue, model string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	targetCount := m.Cfg.ShortMemoryMessagesCount
	thresholdCount := m.Cfg.ShortMemoryMessagesCount + m.Cfg.SummaryMemoryStep

	if m.Tokens.MessageCount < thresholdCount {
		return nil
	}

	systemPrompt := strings.TrimSpace(m.Cfg.PromtMemorySummary)
	if systemPrompt == "" {
		systemPrompt = "Long-term conversation memory:"
	}

	msg := []models.Message{{Role: "system", Content: systemPrompt}}
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

	respLLM, err := enqueueSummaryFn(Queue, llmcalls.QueueItem{Body: models.RequestBody{
		Model:       model,
		Messages:    msg,
		ToolsChoise: "required",
		Tools:       tools.GetToolLongTerm(),
	}})
	if err != nil {
		return err
	}
	if err := detectChosenToolFn(&m.Tools, respLLM); err != nil {
		return err
	}

	m.Logger.Memory("SummaryShortMemory: summarized short-term memory and updated long-term memory")
	return nil
}
