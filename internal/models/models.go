package models

import (
	chatmodels "github.com/nikaydo/personal-assistant/internal/models/chat"
	openroutermodels "github.com/nikaydo/personal-assistant/internal/models/openrouter"
	toolmodels "github.com/nikaydo/personal-assistant/internal/models/tool"
)

type Tool = toolmodels.Tool
type Function = toolmodels.Function
type ToolCall = toolmodels.ToolCall
type ToolFunction = toolmodels.ToolFunction
type ToolFunctionParseResponse = toolmodels.ToolFunctionParseResponse

type Message = chatmodels.Message

type RequestBody = openroutermodels.RequestBody
type Provider = openroutermodels.Provider
type PreferedMinThroughput = openroutermodels.PreferedMinThroughput
type ResponseBody = openroutermodels.ResponseBody
type Model = openroutermodels.Model
type ModelPricing = openroutermodels.ModelPricing
type ModelArchitecture = openroutermodels.ModelArchitecture
type TopProvider = openroutermodels.TopProvider
type DefaultParameters = openroutermodels.DefaultParameters
type ModelData = openroutermodels.ModelData
type EmbendingResponse = openroutermodels.EmbendingResponse
type ToolsChoise = openroutermodels.ToolsChoise
type ToolsChoisePayload = openroutermodels.ToolsChoisePayload

type Choices = openroutermodels.Choices
type Usage = openroutermodels.Usage
type PromptTokensDetails = openroutermodels.PromptTokensDetails
type CostDetails = openroutermodels.CostDetails
type CompletionTokensDetails = openroutermodels.CompletionTokensDetails

func ExtractJSON(s string) (string, error) {
	return openroutermodels.ExtractJSON(s)
}
