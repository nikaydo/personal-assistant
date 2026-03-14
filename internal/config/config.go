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
	PinecoreApiKey     string `json:"pinecore_api_key"`
	PinecoreIndexName  string `json:"pinecore_indexName"`
	PinecoreCloud      string `json:"pinecore_cloud"`
	PinecoreRegion     string `json:"pinecore_region"`
	PinecoreEmbedModel string `json:"pinecore_embedModel"`
	LocalVectorDim     int    `json:"local_vector_dimension"`
	LocalPostgresDSN   string `json:"local_postgres_dsn"`
	LocalPostgresTable string `json:"local_postgres_table"`

	//Config for context
	ContextLimit             int     `json:"context_limit"`
	ContextSavedForResponse  int     `json:"context_saved_for_response"`
	SummaryMemoryStep        int     `json:"summary_memory_step"`
	ContextCoeff             float32 `json:"context_coeff"`
	ContextCoeffCount        int     `json:"context_coeff_count"`
	SystemMemoryPercent      int     `json:"system_memory_percent"`
	UserProfilePercent       int     `json:"user_profile_percent"`
	ToolsMemoryPercent       int     `json:"tools_memory_percent"`
	LongTermPercent          int     `json:"long_term_percent"`
	ShortTermPercent         int     `json:"short_term_percent"`
	SystemPromptPercent      int     `json:"system_prompt_percent"`
	ShortMemoryMessagesCount int     `json:"short_memory_messages_count"`
	MemoryStateFile          string  `json:"memory_state_file"`
	//Api config
	ApiHost string `json:"api_host"`
	ApiPort int    `json:"api_port"`

	//Promts
	PromtSystemChat string `json:"promt_system_chat"`
	// system prompt used specifically when the agent enters reasoning mode.
	PromtSystemAgent             string `json:"promt_system_agent"`
	PromtMemorySummary           string `json:"promt_memory_summary"`
	MemorySummaryUserPromt       string `json:"memory_summary_user_promt"`
	DynamicPromptEnabled         bool   `json:"dynamic_prompt_enabled"`
	DynamicPromptFallbackEnabled bool   `json:"dynamic_prompt_fallback_enabled"`
	DynamicPromptBudgetPercent   int    `json:"dynamic_prompt_budget_percent"`

	// LLM request stability tuning
	LLMRetryMaxAttempts int `json:"llm_retry_max_attempts"`
	LLMRetryBaseDelayMs int `json:"llm_retry_base_delay_ms"`
	LLMRetryMaxDelayMs  int `json:"llm_retry_max_delay_ms"`
}

func ConfigRead(path string) (*Config, error) {
	config := Config{
		DynamicPromptEnabled:         true,
		DynamicPromptFallbackEnabled: true,
		DynamicPromptBudgetPercent:   20,
	}
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
	config.PinecoreApiKey = getEnvString("PINECORE_API_KEY", config.PinecoreApiKey)
	config.PinecoreIndexName = getEnvString("PINECORE_INDEX_NAME", config.PinecoreIndexName)
	config.PinecoreCloud = getEnvString("PINECORE_CLOUD", config.PinecoreCloud)
	config.PinecoreRegion = getEnvString("PINECORE_REGION", config.PinecoreRegion)
	config.PinecoreEmbedModel = getEnvString("PINECORE_EMBED_MODEL", config.PinecoreEmbedModel)
	config.LocalPostgresDSN = getEnvString("LOCAL_POSTGRES_DSN", config.LocalPostgresDSN)
	config.LocalPostgresTable = getEnvString("LOCAL_POSTGRES_TABLE", config.LocalPostgresTable)

	if config.LocalVectorDim, err = getEnvInt("LOCAL_VECTOR_DIMENSION", config.LocalVectorDim); err != nil {
		return err
	}

	if config.ContextLimit, err = getEnvInt("CONTEXT_LIMIT", config.ContextLimit); err != nil {
		return err
	}
	if config.ContextSavedForResponse, err = getEnvInt("CONTEXT_SAVED_FOR_RESPONSE", config.ContextSavedForResponse); err != nil {
		return err
	}
	if config.SummaryMemoryStep, err = getEnvInt("SUMMARY_MEMORY_STEP", config.SummaryMemoryStep); err != nil {
		return err
	}
	config.MemoryStateFile = getEnvString("MEMORY_STATE_FILE", config.MemoryStateFile)

	config.ApiHost = getEnvString("API_HOST", config.ApiHost)
	if config.ApiPort, err = getEnvInt("API_PORT", config.ApiPort); err != nil {
		return err
	}

	config.PromtSystemChat = getEnvString("PROMT_SYSTEM_CHAT", config.PromtSystemChat)
	config.PromtSystemAgent = getEnvString("PROMT_SYSTEM_AGENT", config.PromtSystemAgent)
	config.PromtMemorySummary = getEnvString("PROMT_MEMORY_SUMMARY", config.PromtMemorySummary)
	config.MemorySummaryUserPromt = getEnvString("MEMORY_SUMMARY_USER_PROMT", config.MemorySummaryUserPromt)
	if config.DynamicPromptEnabled, err = getEnvBool("DYNAMIC_PROMPT_ENABLED", config.DynamicPromptEnabled); err != nil {
		return err
	}
	if config.DynamicPromptFallbackEnabled, err = getEnvBool("DYNAMIC_PROMPT_FALLBACK_ENABLED", config.DynamicPromptFallbackEnabled); err != nil {
		return err
	}
	if config.DynamicPromptBudgetPercent, err = getEnvInt("DYNAMIC_PROMPT_BUDGET_PERCENT", config.DynamicPromptBudgetPercent); err != nil {
		return err
	}
	if config.DynamicPromptBudgetPercent <= 0 {
		config.DynamicPromptBudgetPercent = 20
	}

	if config.LLMRetryMaxAttempts, err = getEnvInt("LLM_RETRY_MAX_ATTEMPTS", config.LLMRetryMaxAttempts); err != nil {
		return err
	}
	if config.LLMRetryBaseDelayMs, err = getEnvInt("LLM_RETRY_BASE_DELAY_MS", config.LLMRetryBaseDelayMs); err != nil {
		return err
	}
	if config.LLMRetryMaxDelayMs, err = getEnvInt("LLM_RETRY_MAX_DELAY_MS", config.LLMRetryMaxDelayMs); err != nil {
		return err
	}

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

func getEnvBool(name string, fallback bool) (bool, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, fmt.Errorf("invalid bool env %s=%q: %w", name, value, err)
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
