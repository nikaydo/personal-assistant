package memory

import (
	"strings"

	"github.com/nikaydo/personal-assistant/internal/agent"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/models"
)

var enqueueSummaryFn = func(q *llmcalls.Queue, item llmcalls.QueueItem) (models.ResponseBody, error) {
	return q.AddToQueue(item)
}

var detectChosenToolFn = func(t *agent.Agent, body models.ResponseBody) error {
	_, err := t.DetectChosenTool(body, nil, nil, []models.Message{})
	return err
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
	consumeSnapshot := make([]models.History, 0, len(m.ShortTerm))
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
		Tools:       agent.GetToolLongTerm(),
	}})
	if err != nil {
		return err
	}
	if len(respLLM.Choices) > 0 && len(respLLM.Choices[0].Message.ToolCalls) > 0 {
		if err := detectChosenToolFn(&m.Agent, respLLM); err != nil {
			return err
		}
	} else {
		m.Logger.Memory("SummaryShortMemory: no tool calls in summarization response, skipping tool execution")
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

func countMatchingHistoryPrefix(current, snapshot []models.History) int {
	maxCount := min(len(current), len(snapshot))
	for i := 0; i < maxCount; i++ {
		if current[i] != snapshot[i] {
			return i
		}
	}
	return maxCount
}
