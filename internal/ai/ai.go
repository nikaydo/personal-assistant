package ai

import (
	"fmt"
	"slices"

	"github.com/nikaydo/personal-assistant/internal/agent"
	"github.com/nikaydo/personal-assistant/internal/ai/memory"
	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/database"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
)

var getModelDataFn = llmcalls.GetModelData

type Ai struct {
	Model     []string
	ModelData []models.Model

	Memory *memory.Memory

	Agent agent.Agent

	Queue *llmcalls.Queue

	Config config.Config

	Logger *logg.Logger
}

func Init(config config.Config, aiLog *logg.Logger, db *database.Database) *Ai {
	queueLog := aiLog.WithModule("QUEUE")
	queue := llmcalls.NewQueue(config, 64, queueLog)
	queue.QueueStart()
	agent := agent.Agent{Steps: 10, Model: config.ModelOpenRouter[0], Queue: queue, Dbase: db, Cfg: config}
	mem := &memory.Memory{
		DBase:        db,
		Cfg:          config,
		Logger:       aiLog,
		Agent:        agent,
		SystemMemory: &models.SystemSettings{},
		ToolsMemory:  &[]models.ToolsHistory{},
		Tokens: memory.ContextTokens{
			ContextCoeff:      []float32{config.ContextCoeff},
			ContextCoeffCount: config.ContextCoeffCount,
		},
	}
	if err := mem.LoadState(config.MemoryStateFile); err != nil {
		aiLog.Warn("Init: failed to load memory state, starting with empty state", "error", err)
	}

	return &Ai{
		Queue:  queue,
		Memory: mem,
		Config: config,
		Agent:  agent,
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

func (ai *Ai) GetModelData() error {
	Model, err := getModelDataFn(ai.Config, ai.Logger)
	if err != nil {
		ai.Logger.Error("GetModelData failed", "error", err)
		return err
	}

	ai.Model = ai.Model[:0]
	ai.ModelData = ai.ModelData[:0]
	minAvailableContext := 0
	for _, v := range Model.Data {
		for _, i := range ai.Config.ModelOpenRouter {
			if v.Id == i {
				ai.Logger.Info("Model found", "model", v.Id)
				ai.Model = append(ai.Model, v.Id)
				ai.ModelData = append(ai.ModelData, v)
				availableContext := v.ContextLength - ai.Config.ContextSavedForResponse
				if availableContext <= 0 {
					ai.Logger.Warn("Model has non-positive available context", "model", v.Id, "context_length", v.ContextLength, "reserved_for_response", ai.Config.ContextSavedForResponse)
					continue
				}
				if minAvailableContext == 0 || availableContext < minAvailableContext {
					minAvailableContext = availableContext
				}
			}
		}
	}

	for _, i := range ai.Config.ModelOpenRouter {
		if !slices.Contains(ai.Model, i) {
			ai.Logger.Warn("Model does not support tool or tool_choice", "model", i)
		}
	}

	if len(ai.ModelData) == 0 {
		err := fmt.Errorf("models not found: configure settings.json")
		ai.Logger.Error(err)
		return err
	}

	contextLimit := minAvailableContext
	if ai.Config.ContextLimit > 0 {
		contextLimit = ai.Config.ContextLimit
		if minAvailableContext > 0 && contextLimit > minAvailableContext {
			contextLimit = minAvailableContext
		}
	}
	if contextLimit <= 0 {
		err := fmt.Errorf("context limit is non-positive after model selection")
		ai.Logger.Error(err, "context_limit", contextLimit, "min_available_context", minAvailableContext, "configured_context_limit", ai.Config.ContextLimit)
		return err
	}

	ai.Memory.Tokens.ContextLimit = contextLimit
	return nil
}
