package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	//Config for openrouter
	ApiKeyOpenrouter string   `json:"api_key_openrouter"`
	ModelOpenRouter  []string `json:"model_chat_openrouter"`
	ModelEmbending   string   `json:"model_embending_openrouter"`
	ApiUrlOpenrouter string   `json:"api_url_openrouter"`

	//Config for context
	MaxContextSize       int `json:"max_tokens_context"`
	HighBorderMaxContext int `json:"high_border_max_context"`
	SummaryMemoryStep    int `json:"summary_memory_step"`
	DivisionCoefficient  int `json:"division_coefficient"`

	//Config for Jira
	JiraApiKey      string `json:"jira_api_key"`
	JiraEmail       string `json:"jira_email"`
	JiraPersonalUrl string `json:"jira_personal_url"`

	//Api config
	ApiHost string `json:"api_host"`
	ApiPort int    `json:"api_port"`

	//Promts
	PromtSystemChat        string `json:"promt_system_chat"`
	PromtMemorySummary     string `json:"promt_memory_summary"`
	MemorySummaryUserPromt string `json:"memory_summary_user_promt"`
}

func ConfigRead(path string) (*Config, error) {
	var config Config
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
