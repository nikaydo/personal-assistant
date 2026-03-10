package memory

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/nikaydo/personal-assistant/internal/agent"
	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/database"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
)

const longTermTopK = 20

var createEmbeddingFn = llmcalls.CreateEmbending
var searchByVectorFn = func(db *database.Database, vector []float32, topK int) ([]models.SummarizeResponse, error) {
	results, err := db.SearchByVector(vector, topK)
	if err != nil {
		return nil, err
	}
	out := make([]models.SummarizeResponse, 0, len(results))
	for _, result := range results {
		out = append(out, result.Data)
	}
	return out, nil
}

type Memory struct {
	//системная память
	SystemMemory *models.SystemSettings
	//информация о пользователе
	UserProfile []models.Message
	//tools memory - информация вызванных функциях
	ToolsMemory *[]models.ToolsHistory
	//краткосрочная память
	ShortTerm []models.History

	Agent  agent.Agent
	Tokens ContextTokens

	DBase *database.Database

	Logger *logg.Logger
	Cfg    config.Config

	mu sync.RWMutex
}

func (m *Memory) Memory(question string, answer models.ResponseBody, Queue *llmcalls.Queue, model string) {
	// сохраняем в краткосрочной памяти вопрос и ответ
	m.FillShortMemory(question, answer)
	if err := m.SummaryShortMemory(Queue, model); err != nil {
		m.Logger.Error("SummaryShortMemory: failed to summarize short-term memory:", err)
	}
	// рассчитываем коэффициент контекста
	m.mu.RLock()
	symbolsInContext := m.Tokens.CountSymbolsInContext
	m.mu.RUnlock()
	m.Tokens.ContextCoeffCalc(symbolsInContext, answer)
	m.Logger.Memory("ContextCoeffCalc: calculated context coefficient", "context_coeff", fmt.Sprintf("%v/[%v]", m.Tokens.GetContextCoeff(), m.Tokens.ContextCoeffSnapshot()), "total_tokens", answer.Usage.TotalTokens, "symbols_in_context", symbolsInContext)
	if err := m.SaveState(""); err != nil {
		if m.Logger != nil {
			m.Logger.Warn("Memory: failed to persist memory state", "error", err)
		}
	}
}

func (m *Memory) MessageWithHistory(q, systemPrompt string) []models.Message {
	longTermContent, longTermMsgTokens, longTermSymbols := m.prepareLongTermMemoryMessage(q)

	m.mu.Lock()
	defer m.mu.Unlock()
	messages := []models.Message{}
	m.Tokens.CountSymbolsInContext = 0
	var systemPromptTokens, longTermTokens, shortTermTokens, toolsmemoryTokens *int = new(int), new(int), new(int), new(int)
	//добавляем в историю системный промт и персоанализацию
	messages = m.SystemPromptFill(systemPrompt, messages, systemPromptTokens)
	//добавляем в историю сообщения из информации о пользователе
	messages = m.UserProfileFill(messages)
	//добавляем в историю сообщения из истории выполнения функций
	messages = m.ToolsMemoryFill(messages, toolsmemoryTokens)
	//добавляем в историю сообщения из долгосрочной памяти
	if longTermContent != "" {
		messages = append(messages, models.Message{
			Role:    "system",
			Content: longTermContent,
		})
		*longTermTokens = longTermMsgTokens
		m.Tokens.CountSymbolsInContext += longTermSymbols
	}
	//добавляем в историю сообщения из краткосрочной памяти
	messages = m.ShortMemoryFill(messages, shortTermTokens)
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
	m.Logger.Memory("MessageWithHistory: prepared messages with history for LLM request",
		"system_prompt_tokens", *systemPromptTokens,
		"long_term_tokens", *longTermTokens,
		"short_term_tokens", *shortTermTokens,
		"question_tokens", questionTokens,
		"tools_memory_Tokens", toolsmemoryTokens,
		"total_context_tokens", *systemPromptTokens+*longTermTokens+*shortTermTokens+questionTokens+*toolsmemoryTokens,
		"context_limit", m.Tokens.ContextLimit,
		"context_coeff", m.Tokens.ContextCoeffSnapshot(),
	)
	m.Logger.Info("Memory with history", "messages_count", len(messages))
	return messages
}

func (m *Memory) prepareLongTermMemoryMessage(question string) (string, int, int) {
	if m.Tokens.LongTermLimit == 0 {
		m.Logger.Memory("LongTermMemoryFill: long-term memory is disabled, skipping long-term memory in context")
		return "", 0, 0
	}
	if question == "" {
		m.Logger.Memory("LongTermMemoryFill: question is empty, skipping long-term memory in context")
		return "", 0, 0
	}
	if m.DBase == nil {
		m.Logger.Memory("LongTermMemoryFill: database is nil, skipping long-term memory in context")
		return "", 0, 0
	}

	emb, err := createEmbeddingFn(question, m.Cfg)
	if err != nil {
		m.Logger.Error("LongTermMemoryFill: failed to create embedding, skipping long-term memory:", err)
		return "", 0, 0
	}
	if len(emb.Data) == 0 || len(emb.Data[0].Embedding) == 0 {
		m.Logger.Memory("LongTermMemoryFill: embedding response is empty, skipping long-term memory in context")
		return "", 0, 0
	}

	summaries, err := searchByVectorFn(m.DBase, emb.Data[0].Embedding, longTermTopK)
	if err != nil {
		m.Logger.Error("LongTermMemoryFill: failed to search long-term memory, skipping long-term memory:", err)
		return "", 0, 0
	}
	content, tokens, symbols := m.buildLongTermBlock(summaries)
	if content == "" {
		m.Logger.Memory("LongTermMemoryFill: no long-term memory matched context limits")
		return "", 0, 0
	}
	return content, tokens, symbols
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
	pref := m.SystemMemoryFill()

	m.Tokens.CountSymbolsInContext += len(systemPrompt) + len(pref)
	*systemPromptTokens = int(float32(len(systemPrompt)) / m.Tokens.GetContextCoeff())

	msg = append(msg, models.Message{
		Role:    "system",
		Content: systemPrompt + pref,
	})

	return msg
}

func (m *Memory) SystemMemoryFill() string {
	if m.Tokens.SystemMemoryLimit == 0 {
		m.Logger.Memory("ToolsMemoryFill: tools memory is disabled, skipping tools memory in context")
		return ""
	}

	var str strings.Builder
	str.WriteString("")
	startStr := "\nPersoanalization settings: "
	args := reflect.ValueOf(m.SystemMemory).Elem()
	t := args.Type()

	for i := 0; i < args.NumField(); i++ {
		srcField := args.Field(i)

		if srcField.Kind() == reflect.String && srcField.String() != "" {
			fmt.Fprintf(&str, "%s: %s. ", t.Field(i).Name, srcField.String())
		}
	}
	if str.Len() == 0 {
		return ""
	}
	return startStr + str.String()
}
func (m *Memory) UserProfileFill(msg []models.Message) []models.Message {
	if m.Tokens.UserProfileLimit == 0 {
		m.Logger.Memory("UserProfileFill: user profile memory is disabled, skipping user profile in context")
		return msg
	}
	return msg
}

func (m *Memory) ToolsMemoryFill(msg []models.Message, toolmemeoryTokens *int) []models.Message {
	if m.Tokens.ToolsMemoryLimit == 0 {
		m.Logger.Memory("ToolsMemoryFill: tools memory is disabled, skipping tools memory in context")
		return msg
	}
	for _, i := range *m.ToolsMemory {
		data, _ := json.Marshal(i)
		*toolmemeoryTokens += int(float32(len(data)) / m.Tokens.GetContextCoeff())
		msg = append(msg, models.Message{
			Role:      i.Role,
			Type:      i.Type,
			ID:        i.Id,
			CallId:    i.CallId,
			Name:      i.Name,
			Arguments: i.Arguments,
			Output:    i.Output,
			Content:   i.Content,
		})
	}
	return msg
}

func (m *Memory) LongTermMemoryFill(question string, msg []models.Message, longTermTokens *int) []models.Message {
	if m.Tokens.LongTermLimit == 0 {
		m.Logger.Memory("LongTermMemoryFill: long-term memory is disabled, skipping long-term memory in context")
		return msg
	}
	if question == "" {
		m.Logger.Memory("LongTermMemoryFill: question is empty, skipping long-term memory in context")
		return msg
	}
	if m.DBase == nil {
		m.Logger.Memory("LongTermMemoryFill: database is nil, skipping long-term memory in context")
		return msg
	}

	emb, err := createEmbeddingFn(question, m.Cfg)
	if err != nil {
		m.Logger.Error("LongTermMemoryFill: failed to create embedding, skipping long-term memory:", err)
		return msg
	}
	if len(emb.Data) == 0 || len(emb.Data[0].Embedding) == 0 {
		m.Logger.Memory("LongTermMemoryFill: embedding response is empty, skipping long-term memory in context")
		return msg
	}

	summaries, err := searchByVectorFn(m.DBase, emb.Data[0].Embedding, longTermTopK)
	if err != nil {
		m.Logger.Error("LongTermMemoryFill: failed to search long-term memory, skipping long-term memory:", err)
		return msg
	}
	content, tokens, symbols := m.buildLongTermBlock(summaries)
	if content == "" {
		m.Logger.Memory("LongTermMemoryFill: no long-term memory matched context limits")
		return msg
	}

	msg = append(msg, models.Message{
		Role:    "system",
		Content: content,
	})
	*longTermTokens = tokens
	m.Tokens.CountSymbolsInContext += symbols
	return msg
}

func (m *Memory) buildLongTermBlock(summaries []models.SummarizeResponse) (string, int, int) {
	if len(summaries) == 0 {
		return "", 0, 0
	}
	coeff := m.Tokens.GetContextCoeff()
	if coeff <= 0 {
		coeff = 5
	}

	header := "Long-term conversation memory:\n"
	if int(float32(len(header))/coeff) > m.Tokens.LongTermLimit {
		return "", 0, 0
	}

	var b strings.Builder
	b.WriteString(header)
	currentSymbols := len(header)
	added := 0
	for _, summary := range summaries {
		text := strings.TrimSpace(summary.Text)
		if text == "" {
			continue
		}
		line := fmt.Sprintf("%d. %s\n", added+1, text)
		candidateSymbols := currentSymbols + len(line)
		candidateTokens := int(float32(candidateSymbols) / coeff)
		if candidateTokens > m.Tokens.LongTermLimit {
			break
		}
		b.WriteString(line)
		currentSymbols = candidateSymbols
		added++
	}
	if added == 0 {
		return "", 0, 0
	}

	content := strings.TrimRight(b.String(), "\n")
	tokens := int(float32(len(content)) / coeff)
	return content, tokens, len(content)
}

func (m *Memory) ShortMemoryFill(msg []models.Message, ShortTermTokens *int) []models.Message {
	if m.Tokens.ShortTermLimit <= 0 {
		m.Logger.Memory("ShortMemoryFill: short-term memory is disabled, skipping short-term memory in context")
		return msg
	}
	coeff := m.Tokens.GetContextCoeff()
	if coeff <= 0 {
		coeff = 5
	}

	type shortPair struct {
		user      string
		assistant string
		tokens    int
		symbols   int
	}

	selected := make([]shortPair, 0, len(m.ShortTerm))
	totalTokens := 0
	for i := len(m.ShortTerm) - 1; i >= 0; i-- {
		symbols := len(m.ShortTerm[i].Question.Text) + len(m.ShortTerm[i].Answer.Text)
		pairTokens := int(float32(symbols) / coeff)
		if totalTokens+pairTokens > m.Tokens.ShortTermLimit {
			break
		}
		selected = append(selected, shortPair{
			user:      m.ShortTerm[i].Question.Text,
			assistant: m.ShortTerm[i].Answer.Text,
			tokens:    pairTokens,
			symbols:   symbols,
		})
		totalTokens += pairTokens
	}

	for i := len(selected) - 1; i >= 0; i-- {
		msg = append(msg,
			models.Message{
				Role:    "user",
				Content: selected[i].user,
			}, models.Message{
				Role:    "assistant",
				Content: selected[i].assistant,
			})
		*ShortTermTokens += selected[i].tokens
		m.Tokens.CountSymbolsInContext += selected[i].symbols
	}
	return msg
}

func (m *Memory) FillShortMemory(question string, answer models.ResponseBody) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ShortTerm = append(m.ShortTerm, models.History{
		Question: models.ShotTermQuestion{Text: question},
		Answer:   models.ShotTermAnswer{Text: answer.Choices[0].Message.Content, Usage: answer.Usage},
		Model:    answer.Model,
		Id:       answer.ID,
		Created:  answer.Created,
	})
	m.Tokens.MessageCount++
}
