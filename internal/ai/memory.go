package ai

import (
	mod "github.com/nikaydo/jira-filler/internal/models"
)

func (ai *Ai) HistoryMessage(q mod.Message) []mod.Message {
	msg := ai.makeMessage(ai.Config.PromtSystemChat)
	msg = append(msg, q)
	return msg
}

func (ai *Ai) makeMessage(PromtSystemChat string) []mod.Message {
	msg := []mod.Message{{
		Role:    "system",
		Content: PromtSystemChat,
	}}
	for _, v := range ai.Memory.History {
		msg = append(msg,
			mod.Message{Role: "user", Content: v.Q},
			mod.Message{Role: "assistant", Content: v.A})
	}
	return msg
}
