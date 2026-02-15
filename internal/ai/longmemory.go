package ai

import (
	"fmt"
	"time"

	"github.com/nikaydo/jira-filler/internal/models"
	mod "github.com/nikaydo/jira-filler/internal/models"
)

type LongTimeMemory struct {
	Id        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Embedding []float64 `json:"embedding,omitempty"`
	Time      time.Time `json:"time"`
}

var LongTimeMemoryList []LongTimeMemory

func (ai *Ai) CheckLimit(resp models.ResponseBody) {
	for range 5 {
		if ai.Context.ContextLeghtMax/ai.Config.DivisionCoefficient <= resp.Usage.TotalTokens {
			ai.Memory.History = ai.Memory.History[1:]
		}
	}

	if ai.Config.SummaryMemoryStep < ai.Context.SummaryMemoryStep {
		msg := []mod.Message{{
			Role:    "system",
			Content: ai.Config.PromtMemorySummary,
		}}
		c := len(ai.Memory.History) - ai.Memory.Count
		for _, i := range ai.Memory.History[c:] {
			msg = append(msg,
				mod.Message{Role: "user", Content: i.Q},
				mod.Message{Role: "assistant", Content: i.A})
		}

		msg = append(msg, mod.Message{Role: "user", Content: ai.Config.MemorySummaryUserPromt})
		_, err := ai.Ask(msg, []mod.Tool{})
		if err != nil {
			ai.Logger.Error(fmt.Sprintf("Error in Ask function: %v", err.Error()))
			return
		}

	}
	ai.Memory.Count++
}

func (ai *Ai) GetEmbending(resp models.ResponseBody) {

}
