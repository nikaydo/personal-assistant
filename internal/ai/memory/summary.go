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
	targetCount := m.Cfg.ShortMemoryMessagesCount
	thresholdCount := m.Cfg.ShortMemoryMessagesCount + m.Cfg.SummaryMemoryStep

	if m.Tokens.MessageCount < thresholdCount {
		m.mu.Unlock()
		return nil
	}

	systemPrompt := strings.TrimSpace(m.Cfg.PromtMemorySummary)
	if systemPrompt == "" {
		systemPrompt = "Long-term conversation memory:"
	}

	msg := []models.Message{{Role: "system", Content: systemPrompt}}
	tempCount := m.Tokens.MessageCount
	consumeCount := 0
	consumeSnapshot := make([]History, 0, len(m.ShortTerm))
	for tempCount > targetCount && consumeCount < len(m.ShortTerm) {
		h := m.ShortTerm[consumeCount]
		msg = append(msg,
			models.Message{
				Role:    "user",
				Content: h.Question.Text,
			}, models.Message{
				Role:    "assistant",
				Content: h.Answer.Text,
			})
		consumeSnapshot = append(consumeSnapshot, h)
		consumeCount++
		tempCount--
	}
	m.mu.Unlock()

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

	m.mu.Lock()
	defer m.mu.Unlock()
	commitCount := countMatchingHistoryPrefix(m.ShortTerm, consumeSnapshot)
	if commitCount > 0 {
		m.ShortTerm = m.ShortTerm[commitCount:]
		m.Tokens.MessageCount = max(m.Tokens.MessageCount-commitCount, 0)
	}

	m.Logger.Memory("SummaryShortMemory: summarized short-term memory and updated long-term memory")
	return nil
}

func countMatchingHistoryPrefix(current, snapshot []History) int {
	maxCount := min(len(current), len(snapshot))
	for i := 0; i < maxCount; i++ {
		if current[i] != snapshot[i] {
			return i
		}
	}
	return maxCount
}
