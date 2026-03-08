package ai

import (
	"slices"

	"github.com/nikaydo/personal-assistant/internal/ai/memory"
	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/database"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
)

type Ai struct {
	Model     []string
	ModelData []models.Model

	Memory *memory.Memory

	Queue *llmcalls.Queue

	Config config.Config

	Logger *logg.Logger
}

func Init(config config.Config, aiLog *logg.Logger, db *database.Database) *Ai {
	queueLog := aiLog.WithModule("QUEUE")
	queue := llmcalls.NewQueue(config, 64, queueLog)
	queue.QueueStart()

	return &Ai{
		Queue: queue,
		Memory: &memory.Memory{
			DBase:  db,
			Cfg:    config,
			Logger: aiLog,
			Tokens: memory.ContextTokens{
				ContextCoeff: []float32{config.ContextCoeff},
			},
		},
		Config: config,
		Logger: aiLog,
	}
}

func (ai *Ai) makeBody(messages []models.Message, tools []models.Tool) models.RequestBody {
	body := models.RequestBody{
		Model:       ai.Model[0],
		Messages:    messages,
		ToolsChoise: "auto",
	}
	if len(ai.Model) > 1 {
		body.Models = ai.Model[1:]
	}
	if len(tools) > 0 {
		body.Tools = tools
	}
	return body
}

func (ai *Ai) GetModelData() {
	Model, err := llmcalls.GetModelData(ai.Config, ai.Logger)
	if err != nil {
		ai.Logger.Error("GetModelData failed", "error", err)
		return
	}

	for _, v := range Model.Data {
		for _, i := range ai.Config.ModelOpenRouter {
			if v.Id == i {
				ai.Logger.Info("Model found", "model", v.Id)
				ai.Model = append(ai.Model, v.Id)
				ai.ModelData = append(ai.ModelData, v)
				if v.ContextLength-ai.Config.ContextSavedForResponse < ai.Memory.Tokens.ContextLimit || ai.Memory.Tokens.ContextLimit == 0 {
					ai.Memory.Tokens.ContextLimit = v.ContextLength - ai.Config.ContextSavedForResponse
				}
			}
		}
	}

	for _, i := range ai.Config.ModelOpenRouter {
		if !slices.Contains(ai.Model, i) {
			ai.Logger.Warn("Model does not support tool or tool_choice", "model", i)
		}
	}
	if ai.Memory.Tokens.ContextLimit -= ai.Config.ContextSavedForResponse; ai.Memory.Tokens.ContextLimit < 0 {
		ai.Logger.Warn("Context limit is less than zero after adjustment, setting to zero", "context_limit", ai.Memory.Tokens.ContextLimit)
		ai.Memory.Tokens.ContextLimit = 0
	} else {
		ai.Memory.Tokens.ContextLimit -= ai.Config.ContextSavedForResponse
	}

	if len(ai.ModelData) == 0 {
		ai.Logger.Error("Models not found. Configure settings.json")
		return
	}
}
