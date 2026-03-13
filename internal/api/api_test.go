package api

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	aimodel "github.com/nikaydo/personal-assistant/internal/ai"
	"github.com/nikaydo/personal-assistant/internal/ai/memory"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
)

func TestShutdown_WaitsForMemoryCommitAndFlushesState(t *testing.T) {
	logger := &logg.Logger{
		Customlogger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	stateFile := filepath.Join(t.TempDir(), "memory_state.json")
	mem := &memory.Memory{
		Logger:       logger,
		Cfg:          aimodel.Ai{}.Config,
		SystemMemory: &models.SystemSettings{},
		ToolsMemory:  &[]models.ToolsHistory{},
		Tokens: memory.ContextTokens{
			ContextCoeff: []float32{1},
		},
	}
	mem.Cfg.MemoryStateFile = stateFile
	mem.Cfg.ShortMemoryMessagesCount = 10
	mem.Cfg.SummaryMemoryStep = 10

	answer := models.ResponseBody{
		Model: "model",
		Choices: []models.Choices{
			{Message: models.Message{Content: "answer"}},
		},
	}
	if ok := mem.CommitAsync("question", answer, nil, "model"); !ok {
		t.Fatalf("CommitAsync unexpectedly returned false")
	}

	api := &API{
		Ai: &aimodel.Ai{
			Memory: mem,
		},
	}
	if err := api.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	loaded := &memory.Memory{
		Logger:       logger,
		Cfg:          mem.Cfg,
		SystemMemory: &models.SystemSettings{},
		ToolsMemory:  &[]models.ToolsHistory{},
		Tokens: memory.ContextTokens{
			ContextCoeff: []float32{1},
		},
	}
	if err := loaded.LoadState(""); err != nil {
		t.Fatalf("LoadState returned error: %v", err)
	}
	if len(loaded.ShortTerm) != 1 || loaded.ShortTerm[0].Question.Text != "question" {
		t.Fatalf("unexpected persisted short-term: %#v", loaded.ShortTerm)
	}
}
