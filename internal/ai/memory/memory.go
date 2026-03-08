package memory

import (
	"fmt"
	"sync"

	"github.com/nikaydo/personal-assistant/internal/ai/tools"
	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/database"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
)

var mem string = "Long-term conversation memory:"

type Memory struct {
	//системная память
	SystemMemory string
	//информация о пользователе
	UserProfile []History
	//tools memory - информация о доступных инструментах и их состоянии
	ToolsMemory []History
	//долгосрочная память
	LongTerm []History
	//краткосрочная память
	ShortTerm []History

	Tools tools.Tool

	Tokens ContextTokens

	DBase *database.Database

	Logger *logg.Logger
	Cfg    config.Config

	mu sync.RWMutex
}

type History struct {
	Question ShotTermQuestion `json:"question"`
	Answer   ShotTermAnswer   `json:"answer"`
	Model    string           `json:"model"`
	Id       string           `json:"id"`
	Created  int64            `json:"created"`
}

type ShotTermQuestion struct {
	Text string
}

type ShotTermAnswer struct {
	Text  string
	Usage models.Usage
}

func (m *Memory) Memory(question string, answer models.ResponseBody, Queue *llmcalls.Queue, model string) {
	// сохраняем в краткосрочной памяти вопрос и ответ
	m.FillShortMemory(question, answer)
	if err := m.SummaryShortMemory(mem, Queue, model); err != nil {
		m.Logger.Error("SummaryShortMemory: failed to summarize short-term memory:", err)
	}
	// рассчитываем коэффициент контекста
	m.Tokens.ContextCoeffCalc(question, answer)
	m.Logger.Memory("ContextCoeffCalc: calculated context coefficient", "context_coeff", fmt.Sprintf("%v/[%v]", m.Tokens.GetContextCoeff(), m.Tokens.ContextCoeff), "total_tokens", answer.Usage.TotalTokens, "symbols_in_context", m.Tokens.CountSymbolsInContext)
}

func (m *Memory) MessageWithHistory(q, systemPrompt string) []models.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	messages := []models.Message{}
	m.Tokens.CountSymbolsInContext = 0
	var systemPromptTokens, ShortTermTokens *int = new(int), new(int)
	//добавляем в историю системный промт
	messages = m.SystemPromptFill(systemPrompt, messages, systemPromptTokens)
	//добавляем в историю сообщения из системной памяти
	messages = m.SystemMemoryFill(messages)
	//добавляем в историю сообщения из информации о пользователе
	messages = m.UserProfileFill(messages)
	//добавляем в историю сообщения из истории выполнения функций
	messages = m.ToolsMemoryFill(messages)
	//добавляем в историю сообщения из долгосрочной памяти
	messages = m.LongMemoryFill(messages)
	//добавляем в историю сообщения из краткосрочной памяти
	messages = m.ShortMemoryFill(messages, ShortTermTokens)
	var questionTokens int
	//добавляем в историю текущий вопрос
	if q != "" {
		messages = append(messages, models.Message{
			Role:    "user",
			Content: q,
		})
		m.Tokens.CountSymbolsInContext += len(q)
		questionTokens = int(float32(len(q)) / m.Tokens.GetContextCoeff())
	}
	m.Logger.Memory("MessageWithHistory: prepared messages with history for LLM request", "system_prompt_tokens", systemPromptTokens, "short_term_tokens", ShortTermTokens, "question_tokens", questionTokens, "total_context_tokens", *systemPromptTokens+*ShortTermTokens+questionTokens, "context_limit", m.Tokens.ContextLimit, "context_coeff", m.Tokens.ContextCoeff)
	m.Logger.Info("Memory with history", "messages_count", len(messages))
	return messages
}

func (m *Memory) SystemPromptFill(systemPrompt string, msg []models.Message, systemPromptTokens *int) []models.Message {
	if m.Tokens.SystemPromptPercent == 0 {
		m.Logger.Memory("SystemMemoryFill: system memory is disabled, skipping system prompt in context")
		return msg
	}
	if systemPrompt == "" {
		m.Logger.Memory("SystemMemoryFill: system prompt is empty, skipping system prompt in context")
		return msg
	}
	if float32(len(systemPrompt))/m.Tokens.GetContextCoeff() > float32(m.Tokens.SystemPromptPercent) {
		m.Logger.Memory("SystemMemoryFill: system prompt exceeds system prompt percent, skipping system prompt in context", "system_prompt_length", len(systemPrompt), "system_prompt_percent", m.Tokens.SystemPromptPercent, "context_coeff", m.Tokens.GetContextCoeff())
		return msg
	}
	msg = append(msg, models.Message{
		Role:    "system",
		Content: systemPrompt,
	})
	m.Tokens.CountSymbolsInContext += len(systemPrompt)
	*systemPromptTokens = int(float32(len(systemPrompt)) / m.Tokens.GetContextCoeff())
	return msg
}

func (m *Memory) SystemMemoryFill(msg []models.Message) []models.Message {

	return msg
}

func (m *Memory) UserProfileFill(msg []models.Message) []models.Message {
	if m.Tokens.UserProfileLimit == 0 {
		m.Logger.Memory("UserProfileFill: user profile memory is disabled, skipping user profile in context")
		return msg
	}
	return msg
}

func (m *Memory) ToolsMemoryFill(msg []models.Message) []models.Message {
	if m.Tokens.ToolsMemoryLimit == 0 {
		m.Logger.Memory("ToolsMemoryFill: tools memory is disabled, skipping tools memory in context")
		return msg
	}
	return msg
}

func (m *Memory) LongMemoryFill(msg []models.Message) []models.Message {
	if m.Tokens.LongTermLimit == 0 {
		m.Logger.Memory("LongMemoryFill: long-term memory is disabled, skipping long-term memory in context")
		return msg
	}

	return msg
}

func (m *Memory) ShortMemoryFill(msg []models.Message, ShortTermTokens *int) []models.Message {
	for i := range m.ShortTerm {
		msg = append(msg,
			models.Message{
				Role:    "user",
				Content: m.ShortTerm[i].Question.Text,
			}, models.Message{
				Role:    "assistant",
				Content: m.ShortTerm[i].Answer.Text,
			})

		symbolsInMsg := len(m.ShortTerm[i].Question.Text) + len(m.ShortTerm[i].Answer.Text)
		*ShortTermTokens += int(float32(symbolsInMsg) / m.Tokens.GetContextCoeff())
		m.Tokens.CountSymbolsInContext += symbolsInMsg
	}
	return msg
}

func (m *Memory) FillShortMemory(question string, answer models.ResponseBody) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ShortTerm = append(m.ShortTerm, History{
		Question: ShotTermQuestion{Text: question},
		Answer:   ShotTermAnswer{Text: answer.Choices[0].Message.Content, Usage: answer.Usage},
		Model:    answer.Model,
		Id:       answer.ID,
		Created:  answer.Created,
	})
	m.Tokens.MessageCount++
}
