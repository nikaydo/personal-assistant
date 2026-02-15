package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nikaydo/jira-filler/internal/ai"
	"github.com/nikaydo/jira-filler/internal/models"
)

type Query struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

func (api *API) chat(w http.ResponseWriter, r *http.Request) {
	var Query Query
	err := json.NewDecoder(r.Body).Decode(&Query)
	if err != nil {
		api.Ai.Logger.Error(fmt.Sprintf("chat decode failed: %v", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if Query.Message == "" {
		err = fmt.Errorf("message is required")
		api.Ai.Logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msg, err := api.Ai.MakeAsk(models.Message{Role: "user", Content: Query.Message}, ai.GetToolRouter())
	if err != nil {
		api.Ai.Logger.Error(fmt.Sprintf("chat processing failed: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(msg)
	if err != nil {
		api.Ai.Logger.Error(fmt.Sprintf("chat encode failed: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(data)
}
