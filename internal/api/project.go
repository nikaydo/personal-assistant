package api

import (
	"encoding/json"
	"net/http"

	"github.com/nikaydo/jira-filler/internal/ai"
	"github.com/nikaydo/jira-filler/internal/models"
)

func (api *API) CreateProject(w http.ResponseWriter, r *http.Request) {
	var Query Query
	err := json.NewDecoder(r.Body).Decode(&Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	msg, err := api.Ai.MakeAsk(models.Message{Role: "user", Content: Query.Message}, ai.GetToolRouter())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(data)
}
