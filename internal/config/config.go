package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	//Config for openrouter
	ApiKeyOpenrouter           string   `json:"api_key_openrouter"`
	ModelOpenRouter            []string `json:"model_chat_openrouter"`
	ModelEmbending             string   `json:"model_embending_openrouter"`
	ApiUrlOpenrouter           string   `json:"api_url_openrouter"`
	ApiUrlOpenrouterEmbeddings string   `json:"api_url_openrouter_embeddings"`

	//config for database
	DatabaseApiKey string `json:"database_api_key"`
	IndexName      string `json:"indexName"`
	Cloud          string `json:"cloud"`
	Region         string `json:"region"`
	EmbedModel     string `json:"embedModel"`

	//Config for context
	MaxContextSize       int `json:"max_tokens_context"`
	HighBorderMaxContext int `json:"high_border_max_context"`
	SummaryMemoryStep    int `json:"summary_memory_step"`
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

	SpotifyRefresh string `json:"spotify_refresh"`
	SpotifyAccess  string `json:"spotify_access"`
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
	if err := applyEnvOverrides(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func applyEnvOverrides(config *Config) error {
	var err error

	config.ApiKeyOpenrouter = getEnvString("API_KEY_OPENROUTER", config.ApiKeyOpenrouter)
	config.ModelOpenRouter = getEnvStringSlice("MODEL_CHAT_OPENROUTER", config.ModelOpenRouter)
	config.ModelEmbending = getEnvString("MODEL_EMBENDING_OPENROUTER", config.ModelEmbending)
	config.ApiUrlOpenrouter = getEnvString("API_URL_OPENROUTER", config.ApiUrlOpenrouter)
	config.ApiUrlOpenrouterEmbeddings = getEnvString("API_URL_OPENROUTER_EMBEDDINGS", config.ApiUrlOpenrouterEmbeddings)

	if config.MaxContextSize, err = getEnvInt("MAX_TOKENS_CONTEXT", config.MaxContextSize); err != nil {
		return err
	}
	if config.HighBorderMaxContext, err = getEnvInt("HIGH_BORDER_MAX_CONTEXT", config.HighBorderMaxContext); err != nil {
		return err
	}
	if config.SummaryMemoryStep, err = getEnvInt("SUMMARY_MEMORY_STEP", config.SummaryMemoryStep); err != nil {
		return err
	}
	config.JiraApiKey = getEnvString("JIRA_API_KEY", config.JiraApiKey)
	config.JiraEmail = getEnvString("JIRA_EMAIL", config.JiraEmail)
	config.JiraPersonalUrl = getEnvString("JIRA_PERSONAL_URL", config.JiraPersonalUrl)

	config.ApiHost = getEnvString("API_HOST", config.ApiHost)
	if config.ApiPort, err = getEnvInt("API_PORT", config.ApiPort); err != nil {
		return err
	}

	config.PromtSystemChat = getEnvString("PROMT_SYSTEM_CHAT", config.PromtSystemChat)
	config.PromtMemorySummary = getEnvString("PROMT_MEMORY_SUMMARY", config.PromtMemorySummary)
	config.MemorySummaryUserPromt = getEnvString("MEMORY_SUMMARY_USER_PROMT", config.MemorySummaryUserPromt)

	return nil
}

func getEnvString(name, fallback string) string {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}
	return value
}

func getEnvInt(name string, fallback int) (int, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("invalid integer env %s=%q: %w", name, value, err)
	}

	return parsed, nil
}

func getEnvStringSlice(name string, fallback []string) []string {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}

	items := strings.Split(value, ",")
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		filtered = append(filtered, item)
	}

	if len(filtered) == 0 {
		return fallback
	}

	return filtered
}
