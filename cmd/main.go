package main

import (
	"fmt"

	"github.com/nikaydo/personal-assistant/internal/api"
	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/logg"
)

func main() {
	l := logg.InitLogger()
	systemLog := l.WithModule("SYSTEM")
	apiLog := l.WithModule("API")
	aiLog := l.WithModule("AI")

	systemLog.Info("Starting application")
	config, err := config.ConfigRead("./settings.json")
	if err != nil {
		systemLog.Error("Failed to read config:", err)
		return
	}
	systemLog.Info("Config loaded")

	api, err := api.SetupApi(*config, apiLog)
	if err != nil {
		apiLog.Error("Failed to setup api:", err)
		return
	}
	apiLog.Info(fmt.Sprintf("Server configured on addr %s:%d", config.ApiHost, config.ApiPort))

	aiLog.Info("Loading model metadata")
	api.Ai.GetModelData()
	aiLog.Info(fmt.Sprintf("Context length: %d", api.Ai.Memory.Tokens.ContextLimit))

	api.Ai.Memory.Tokens.CalculateContextLimit(api.Ai.Config)
	api.Ai.Logger.Memory("CalculateContextLimit: calculated context limit for each memory type", "system_memory_tokens", api.Ai.Memory.Tokens.SystemMemoryLimit, "user_profile_tokens", api.Ai.Memory.Tokens.UserProfileLimit, "tools_memory_tokens", api.Ai.Memory.Tokens.ToolsMemoryLimit, "long_term_tokens", api.Ai.Memory.Tokens.LongTermLimit, "short_term_tokens", api.Ai.Memory.Tokens.ShortTermLimit)

	api.SetupRoutes()
	apiLog.Info("Routes setup")
	systemLog.Info(fmt.Sprintf("Server starting on addr %s:%d", config.ApiHost, config.ApiPort))
	systemLog.Info("Ready")
	err = api.Start()
	if err != nil {
		systemLog.Error("Server start failed:", err)
		return
	}
}
