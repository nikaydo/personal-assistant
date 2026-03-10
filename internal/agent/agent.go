package agent

import (
	"encoding/json"
	"errors"

	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/database"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
	"github.com/nikaydo/personal-assistant/internal/services"
	command "github.com/nikaydo/personal-assistant/internal/services/command"
)

type Agent struct {
	Steps int
	Model string

	Dbase *database.Database

	Cfg config.Config

	Queue *llmcalls.Queue

	Logger *logg.Logger

	SystemPrompt string

	History *[]models.Message
}

type History struct {
	Type string
	Tool models.ToolsHistory
	Msg  models.Message
}

type AgentResponse struct {
	Thought string `json:"thought"`
	Func    Func   `json:"action"`
}

type Func struct {
	Function  string `json:"function"`
	Arguments string `json:"arguments"`
}

func parseAgentResponse(body models.ResponseBody) (AgentResponse, error) {
	var args AgentResponse
	err := json.Unmarshal([]byte(body.Choices[0].Message.ToolCalls[0].Function.Arguments), &args)
	if err != nil {
		return AgentResponse{}, err
	}
	return args, nil
}

func (a *Agent) Run(body models.ResponseBody) (models.ResponseBody, error) {
	a.Logger.Info("Agent.Run called", "initial_tool", body.Choices[0].Message.ToolCalls[0].Function.Name)
	r, err := parseAgentResponse(body)
	if err != nil {
		a.Logger.Error("parseAgentResponse failed", "error", err)
		return models.ResponseBody{}, err
	}
	// make sure the first thought is never empty (omitempty in struct would drop it)
	safeThought := r.Thought
	if safeThought == "" {
		safeThought = " "
	}
	*a.History = append(*a.History, models.Message{
		Role:    "assistant",
		Content: safeThought})

	for range a.Steps {
		respLLM, err := a.AskLLM("auto")
		if err != nil {
			a.Logger.Error("AskLLM failed", "error", err)
			return models.ResponseBody{}, err
		}
		a.Logger.Info("AskLLM: ", respLLM)
		out, stop, err := a.RunTool(respLLM)
		if err != nil {
			a.Logger.Error("RunTool failed", "error", err)
			return models.ResponseBody{}, err
		}
		a.Logger.Info("RunTool: ", out)
		if stop {
			a.Logger.Info("agent stopping")
			*a.History = append(*a.History, models.Message{
				Role:      "function",
				ID:        respLLM.Choices[0].Message.ID,
				Type:      respLLM.Choices[0].Message.Type,
				Name:      respLLM.Choices[0].Message.ToolCalls[0].Function.Name,
				Arguments: respLLM.Choices[0].Message.ToolCalls[0].Function.Arguments,
				Output:    "final",
				Content:   "call input and output"})
			return a.AskLLM("none")
		}

		if err := a.CollectContext(respLLM, out); err != nil {
			a.Logger.Error("CollectContext failed", "error", err)
			return models.ResponseBody{}, err
		}
		a.Logger.Info("RunTool: ", a.History)
	}
	return models.ResponseBody{}, errors.New("wtf")
}

func (a *Agent) RunTool(body models.ResponseBody) (string, bool, error) {
	for _, i := range body.Choices[0].Message.ToolCalls {
		a.Logger.Info("RunTool processing", "tool", i.Function.Name)
		switch i.Function.Name {
		case "reasoning":
			return "", false, nil
		case "command":
			a.Logger.Info("executing command", "args", i.Function.Arguments)
			// use command service directly for simple invocation
			svc := command.NewService()
			out, err := svc.ExecuteFromLLM(i.Function.Arguments)
			if err != nil {
				a.Logger.Error("command execution failed", "error", err)
				return "", false, err
			}
			return out, false, nil
		case "stop":
			var args struct {
				R string `json:"response"`
			}
			if err := json.Unmarshal([]byte(body.Choices[0].Message.ToolCalls[0].Function.Arguments), &args); err != nil {
				return "", false, errors.New("unknown tool")
			}
			return "", true, nil
		}
	}
	return "", false, errors.New("unknown tool")
}

func (a *Agent) RunFunc(args AgentResponse) (string, error) {
	a.Logger.Info("RunFunc called", "function", args.Func.Function)
	switch args.Func.Function {
	case "command":
		// delegate to the command service which knows how to interpret
		// the arguments string emitted by the LLM.
		svc := services.NewCommandService()
		out, err := svc.ExecuteFromLLM(args.Func.Arguments)
		if err != nil {
			a.Logger.Error("RunFunc command failed", "error", err)
			return "", err
		}
		return out, nil
	default:
		a.Logger.Warn("RunFunc unknown function", "name", args.Func.Function)
		return "", errors.New("unknown func")
	}
}

func getFuncNameArgs(body models.ResponseBody) (struct {
	Name         string
	Args         string
	FinishReason string
}, error) {
	var data struct {
		Name         string
		Args         string
		FinishReason string
	}
	data.Name = body.Choices[0].Message.ToolCalls[0].Function.Name

	data.Args = body.Choices[0].Message.ToolCalls[0].Function.Arguments

	data.FinishReason = body.Choices[0].FinishReason
	return data, nil
}

func (a *Agent) CollectContext(body models.ResponseBody, funcOutput string) error {
	data, err := getFuncNameArgs(body)
	if err != nil {
		a.Logger.Error("CollectContext getFuncNameArgs failed", "error", err, "body", body)
		return err
	}
	args, err := parseAgentResponse(body)
	if err != nil {
		a.Logger.Error("CollectContext parseAgentResponse failed", "error", err)
		return err
	}
	tool := body.Choices[0].Message.ToolCalls[0]

	if data.Name != "reasoning" {
		*a.History = append(*a.History,
			models.Message{
				Role:      "function",
				ID:        tool.ID,
				Type:      tool.Type,
				Name:      data.Name,
				Arguments: data.Args,
				Output:    funcOutput,
				Content:   "call input and output",
			},
		)
	}

	if args.Thought != "" {
		*a.History = append(*a.History,
			models.Message{
				Role:    "assistant",
				Content: args.Thought},
		)
	}

	return nil
}

func (a *Agent) AskLLM(ToolsChoise string) (models.ResponseBody, error) {
	respLLM, err := a.Queue.AddToQueue(llmcalls.QueueItem{Body: models.RequestBody{
		Model:       a.Model,
		Messages:    *a.History,
		ToolsChoise: ToolsChoise,
		Tools:       GetAgentTool(),
	}})
	if err != nil {
		return models.ResponseBody{}, err
	}
	return respLLM, nil
}
