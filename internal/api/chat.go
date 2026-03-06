package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nikaydo/personal-assistant/internal/models"
)

type Query struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

func (api *API) chat(w http.ResponseWriter, r *http.Request) {
	var Query Query
	err := json.NewDecoder(r.Body).Decode(&Query)
	if err != nil {
		api.Ai.Logger.Error("chat decode failed:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if Query.Message == "" {
		err = fmt.Errorf("message is required")
		api.Ai.Logger.Error("chat validation failed:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	api.Ai.Logger.Info("chat request accepted", "message_len", len(Query.Message), "type", Query.Type)

	msg, err := api.Ai.MakeAsk(models.Message{Role: "user", Content: Query.Message}, []models.Tool{})
	if err != nil {
		api.Ai.Logger.Error("chat processing failed:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(msg)
	if err != nil {
		api.Ai.Logger.Error("chat encode failed:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	finishReason := ""
	if len(msg.Choices) > 0 {
		finishReason = msg.Choices[0].FinishReason
	}
	api.Ai.Logger.Info("chat response ready", "finish_reason", finishReason, "total_tokens", msg.Usage.TotalTokens)
	w.Write(data)
}

func (api *API) GetMemory(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(api.Ai.Memory)
	if err != nil {
		api.Ai.Logger.Error("GetMemory marshal failed:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (api *API) GetMessage(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(api.Ai.Memory.HistoryMessage(models.Message{}, "system promt"))
	if err != nil {
		api.Ai.Logger.Error("GetMessage marshal failed:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}
