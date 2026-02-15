package models

import (
	chatmodels "github.com/nikaydo/jira-filler/internal/models/chat"
	jiramodels "github.com/nikaydo/jira-filler/internal/models/jira"
	openroutermodels "github.com/nikaydo/jira-filler/internal/models/openrouter"
	toolmodels "github.com/nikaydo/jira-filler/internal/models/tool"
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

type ProjectSearchOptions = jiramodels.ProjectSearchOptions
type ProjectSearchOptionsScheme = jiramodels.ProjectSearchOptionsScheme

func ExtractJSON(s string) (string, error) {
	return openroutermodels.ExtractJSON(s)
}
