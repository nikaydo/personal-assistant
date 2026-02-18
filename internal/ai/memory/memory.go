package memory

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/database"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
	mod "github.com/nikaydo/personal-assistant/internal/models"
)

/*

пользователь пишет сообщение
ии отвечает
перед следующим сообщение собираеться история диалога 1 кс память (сообщения вопрос ответ user assistant) 2 дс память (релевантные данные по вопросу)

1. из бд на пк пользователя +
1. просто вставляеться в конткст +

2. собираеться кс память без дс памяти и делаеться запрос к llm c принудительным вызовом tool
2. llm возвращяет параметры для получения релевантных данных и делаеться запрос к бд
2. данные из бд вставляються в диалог

переоформление дс памяти происходит раз в 5 сообщений пользователя

*/

type Memory struct {
	//системная память для промта
	SystemMemory string
	//информация о пользователе
	UserProfile string
	//информация о фильтрах и которые можно применить
	SystemMemoryInfo string
	//долгосрочная память
	LongTerm string
	//краткосрочная память
	ShortTerm []ShortTerm

	Count int

	DBase *database.DBase

	Logger *logg.Logger
	Cfg    *config.Config

	mu             sync.RWMutex
	summaryRunning bool
}

type ShortTerm struct {
	UserQuestion  string `json:"user_question"`
	LLMAnswer     string `json:"llm_answer"`
	TotalTokens   int    `json:"total_tokens"`
	CurrentTokens int    `json:"current_tokens"`
}

func (st *Memory) FillShortMemory(q, a string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.ShortTerm = append(st.ShortTerm, ShortTerm{UserQuestion: q, LLMAnswer: a})
}

// недописано
func (mem *Memory) GetEmbending(r mod.ResponseBody) {
	mem.Logger.Memory("GetEmbending start", "model", mem.Cfg.ModelEmbending)
	body := mod.RequestBody{
		Model: mem.Cfg.ModelEmbending,
		Input: r.GetContent(),
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		mem.Logger.Error("GetEmbending: json marshal failed:", err)
		return
	}
	req, err := http.NewRequest("POST", mem.Cfg.ApiUrlOpenrouterEmbeddings, bytes.NewBuffer(jsonBody))
	if err != nil {
		mem.Logger.Error("GetEmbending: create request failed:", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+mem.Cfg.ApiKeyOpenrouter)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		mem.Logger.Error("GetEmbending: request failed:", err)
		return
	}
	defer resp.Body.Close()
	mem.Logger.Memory("GetEmbending response", "status", resp.StatusCode)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		mem.Logger.Error("GetEmbending: read response failed:", err)
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		mem.Logger.Error("GetEmbending unexpected status", "status", resp.StatusCode, "body", string(respBody))
		return
	}
	var embending models.EmbendingResponse
	err = json.Unmarshal(respBody, &embending)
	if err != nil {
		mem.Logger.Error("GetEmbending: unmarshal failed:", err)
		return
	}
	mem.Logger.Memory("GetEmbending parsed", "embeddings", len(embending.Data))

}

func (mem *Memory) HistoryMessage(q mod.Message, promt string) []mod.Message {
	shortTerm := mem.ShortTermSnapshot()
	msg := make([]mod.Message, 0, len(shortTerm)*2+2)
	if promt != "" {
		msg = append(msg, mod.Message{
			Role:    "system",
			Content: promt,
		})
	}
	for _, v := range shortTerm {
		msg = append(msg,
			mod.Message{Role: "user", Content: v.UserQuestion},
			mod.Message{Role: "assistant", Content: v.LLMAnswer})
	}
	return append(msg, q)
}

func (mem *Memory) collectShortMemory(count int, promt string, shortTerm []ShortTerm) []mod.Message {
	msg := []mod.Message{{
		Role:    "system",
		Content: promt,
	}}
	for _, v := range shortTerm[len(shortTerm)-count:] {
		msg = append(msg,
			mod.Message{Role: "user", Content: v.UserQuestion},
			mod.Message{Role: "assistant", Content: v.LLMAnswer})
	}
	return msg
}

func (mem *Memory) ShortTermSnapshot() []ShortTerm {
	mem.mu.RLock()
	defer mem.mu.RUnlock()
	if len(mem.ShortTerm) == 0 {
		return nil
	}
	snapshot := make([]ShortTerm, len(mem.ShortTerm))
	copy(snapshot, mem.ShortTerm)
	return snapshot
}

func (mem *Memory) ShortTermLen() int {
	mem.mu.RLock()
	defer mem.mu.RUnlock()
	return len(mem.ShortTerm)
}

func (mem *Memory) SummaryCounter() int {
	mem.mu.RLock()
	defer mem.mu.RUnlock()
	return mem.Count
}
