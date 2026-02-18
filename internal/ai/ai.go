package ai

import (
	"slices"

	"github.com/nikaydo/personal-assistant/internal/ai/memory"
	"github.com/nikaydo/personal-assistant/internal/config"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
	"github.com/nikaydo/personal-assistant/internal/services"
)

type Memory = memory.Memory

type Ai struct {
	Model     []string
	ModelData []models.Model

	Context Context

	Memory *memory.Memory

	Config config.Config

	Jira *services.JiraService

	ToolConf *ToolConf

	Logger *logg.Logger
}

type Context struct {
	ContextLeghtMax     int
	ContextLeghtCurrent int
	SummaryMemoryStep   int
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
				if v.ContextLength-ai.Config.HighBorderMaxContext < ai.Context.ContextLeghtMax || ai.Context.ContextLeghtMax == 0 {
					ai.Context.ContextLeghtMax = v.ContextLength - ai.Config.HighBorderMaxContext
				}
			}
		}
	}

	for _, i := range ai.Config.ModelOpenRouter {
		if !slices.Contains(ai.Model, i) {
			ai.Logger.Warn("Model does not support tool and tool_choice", "model", i)
		}
	}

	if ai.Config.MaxContextSize != 0 {
		ai.Context.ContextLeghtMax = ai.Config.MaxContextSize - ai.Config.HighBorderMaxContext
	}
	if len(ai.ModelData) == 0 {
		ai.Logger.Error("Models not found. Configure settings.json")
		return
	}
}
