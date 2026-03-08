package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

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
	if err := api.Ai.GetModelData(); err != nil {
		aiLog.Error("Failed to load model metadata:", err)
		return
	}
	aiLog.Info(fmt.Sprintf("Context length: %d", api.Ai.Memory.Tokens.ContextLimit))

	api.Ai.Memory.Tokens.CalculateContextLimit(api.Ai.Config)
	api.Ai.Logger.Memory("CalculateContextLimit: calculated context limit for each memory type", "system_memory_tokens", api.Ai.Memory.Tokens.SystemMemoryLimit, "user_profile_tokens", api.Ai.Memory.Tokens.UserProfileLimit, "tools_memory_tokens", api.Ai.Memory.Tokens.ToolsMemoryLimit, "long_term_tokens", api.Ai.Memory.Tokens.LongTermLimit, "short_term_tokens", api.Ai.Memory.Tokens.ShortTermLimit)

	api.SetupRoutes()
	apiLog.Info("Routes setup")
	systemLog.Info(fmt.Sprintf("Server starting on addr %s:%d", config.ApiHost, config.ApiPort))
	systemLog.Info("Ready")

	startErrCh := make(chan error, 1)
	go func() {
		startErrCh <- api.Start()
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err = <-startErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			systemLog.Error("Server start failed:", err)
		}
		return
	case <-sigCtx.Done():
	}

	systemLog.Info("Shutdown requested, stopping services")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := api.Shutdown(shutdownCtx); err != nil {
		systemLog.Error("Shutdown failed:", err)
	}

	if err = <-startErrCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		systemLog.Error("Server stopped with error:", err)
	}
}
