package ai

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/nikaydo/personal-assistant/internal/agent"
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

func TestMakeAsk_HandlesToolCalls(t *testing.T) {
	oldAdd := addToQueueFn
	oldDetect := detectChosenToolFn
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
									Name:      "agent_mode",
									Arguments: "{\"thought\":\"x\"}",
								},
							},
						},
					},
				},
			},
		}, nil
	}
	detectChosenToolFn = func(_ *agent.Agent, _ models.ResponseBody, _ *models.SystemSettings, _ *[]models.ToolsHistory, _ []models.Message) (models.ResponseBody, error) {
		return models.ResponseBody{
			Choices: []models.Choices{
				{
					Message: models.Message{
						Content: "ok",
					},
				},
			},
		}, nil
	}
	t.Cleanup(func() {
		addToQueueFn = oldAdd
		detectChosenToolFn = oldDetect
	})

	ai := newTestAI(config.Config{})
	ai.Model = []string{"test-model"}
	resp, err := ai.MakeAsk("hello", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Choices[0].Message.Content != "ok" {
		t.Fatalf("unexpected content: %q", resp.Choices[0].Message.Content)
	}
}

func TestInit_LoadsMemoryStateFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory_state.json")
	logger := newTestLogger()

	seed := &memory.Memory{
		Logger: logger,
		Cfg: config.Config{
			MemoryStateFile: path,
		},
		Tokens: memory.ContextTokens{
			ContextCoeff:      []float32{1},
			ContextCoeffCount: 12,
		},
	}
	seed.ShortTerm = []models.History{
		{
			Question: models.ShotTermQuestion{Text: "q1"},
			Answer:   models.ShotTermAnswer{Text: "a1"},
		},
	}
	seed.Tokens.MessageCount = 1
	seed.Tokens.SetContextCoeffSnapshot([]float32{1.5, 2.5})
	if err := seed.SaveState(path); err != nil {
		t.Fatalf("SaveState returned error: %v", err)
	}

	ai := Init(config.Config{
		ContextCoeff:      5,
		ContextCoeffCount: 12,
		MemoryStateFile:   path,
	}, logger, nil)
	t.Cleanup(func() {
		if ai.Queue != nil {
			ai.Queue.Stop()
		}
	})

	if len(ai.Memory.ShortTerm) != 1 {
		t.Fatalf("unexpected restored short-term length: %d", len(ai.Memory.ShortTerm))
	}
	if ai.Memory.ShortTerm[0].Question.Text != "q1" {
		t.Fatalf("unexpected restored short-term content: %#v", ai.Memory.ShortTerm[0])
	}
	if ai.Memory.Tokens.MessageCount != 1 {
		t.Fatalf("unexpected restored message count: %d", ai.Memory.Tokens.MessageCount)
	}
	coeff := ai.Memory.Tokens.ContextCoeffSnapshot()
	if len(coeff) != 2 || coeff[0] != 1.5 || coeff[1] != 2.5 {
		t.Fatalf("unexpected restored context coefficients: %#v", coeff)
	}
}
