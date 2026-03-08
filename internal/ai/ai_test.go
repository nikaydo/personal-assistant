package ai

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/nikaydo/personal-assistant/internal/ai/memory"
	"github.com/nikaydo/personal-assistant/internal/config"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
)

func newTestLogger() *logg.Logger {
	return &logg.Logger{
		Customlogger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func newTestAI(cfg config.Config) *Ai {
	logger := newTestLogger()
	return &Ai{
		Config: cfg,
		Logger: logger,
		Queue:  &llmcalls.Queue{},
		Memory: &memory.Memory{
			Logger: logger,
			Tokens: memory.ContextTokens{
				ContextCoeff:      []float32{1},
				ContextCoeffCount: cfg.ContextCoeffCount,
			},
		},
	}
}

func TestGetModelData_ContextLimitCalculatedOnce(t *testing.T) {
	oldFn := getModelDataFn
	getModelDataFn = func(cfg config.Config, log *logg.Logger) (models.ModelData, error) {
		return models.ModelData{
			Data: []models.Model{
				{Id: "m1", ContextLength: 32000},
				{Id: "m2", ContextLength: 28000},
			},
		}, nil
	}
	t.Cleanup(func() {
		getModelDataFn = oldFn
	})

	ai := newTestAI(config.Config{
		ModelOpenRouter:         []string{"m1", "m2"},
		ContextSavedForResponse: 5000,
	})

	if err := ai.GetModelData(); err != nil {
		t.Fatalf("GetModelData returned error: %v", err)
	}

	if ai.Memory.Tokens.ContextLimit != 23000 {
		t.Fatalf("unexpected context limit: got=%d want=%d", ai.Memory.Tokens.ContextLimit, 23000)
	}
}

func TestGetModelData_ReturnsErrorWhenNoModelsMatched(t *testing.T) {
	oldFn := getModelDataFn
	getModelDataFn = func(cfg config.Config, log *logg.Logger) (models.ModelData, error) {
		return models.ModelData{
			Data: []models.Model{
				{Id: "other-model", ContextLength: 32000},
			},
		}, nil
	}
	t.Cleanup(func() {
		getModelDataFn = oldFn
	})

	ai := newTestAI(config.Config{
		ModelOpenRouter:         []string{"missing-model"},
		ContextSavedForResponse: 1000,
	})

	if err := ai.GetModelData(); err == nil {
		t.Fatalf("expected GetModelData to fail when configured models are missing")
	}
}

func TestMakeAsk_ReturnsNotImplementedOnToolCalls(t *testing.T) {
	oldAdd := addToQueueFn
	addToQueueFn = func(q *llmcalls.Queue, item llmcalls.QueueItem) (models.ResponseBody, error) {
		return models.ResponseBody{
			Choices: []models.Choices{
				{
					Message: models.Message{
						ToolCalls: []models.ToolCall{
							{
								ID:   "tc-1",
								Type: "function",
								Function: models.ToolFunction{
									Name:      "summarize",
									Arguments: "{}",
								},
							},
						},
					},
				},
			},
		}, nil
	}
	t.Cleanup(func() {
		addToQueueFn = oldAdd
	})

	ai := newTestAI(config.Config{})
	ai.Model = []string{"test-model"}
	_, err := ai.MakeAsk("hello", nil)
	if !errors.Is(err, ErrToolCallsNotImplemented) {
		t.Fatalf("expected ErrToolCallsNotImplemented, got %v", err)
	}
}
