package agent

import (
	"encoding/json"
	"errors"
	"fmt"

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
	Thought  string `json:"thought"`
	Question string `json:"question,omitempty"`
	Func     Func   `json:"action"`
}

type Func struct {
	Function  string          `json:"function"`
	Arguments json.RawMessage `json:"arguments"`
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
	if len(body.Choices) == 0 || len(body.Choices[0].Message.ToolCalls) == 0 {
		return models.ResponseBody{}, errors.New("agent: empty tool calls")
	}
	a.Logger.Agent("Agent.Run called", "initial_tool", body.Choices[0].Message.ToolCalls[0].Function.Name)
	r, err := parseAgentResponse(body)
	if err != nil {
		a.Logger.Error("parseAgentResponse failed", "error", err)
		return models.ResponseBody{}, err
	}
	if r.Question != "" {
		*a.History = append(*a.History, models.Message{Role: "user", Content: r.Question})
	}
	// make sure the first thought is never empty (omitempty in struct would drop it)
	safeThought := r.Thought
	if safeThought == "" {
		safeThought = " "
	}
	*a.History = append(*a.History, models.Message{
		Role:    "assistant",
		Content: safeThought})

	// run a fixed number of steps rather than iterating over an integer value
	for i := range a.Steps {
		a.Logger.Agent("agent iteration", "step", i)
		respLLM, err := a.AskLLM("auto")
		if err != nil {
			a.Logger.Error("AskLLM failed", "error", err)
			return models.ResponseBody{}, err
		}
		a.Logger.Agent("AskLLM: ", respLLM)
		out, stop, err := a.RunTool(respLLM)
		if err != nil {
			a.Logger.Error("RunTool failed", "error", err)
			return models.ResponseBody{}, err
		}
		a.Logger.Agent("RunTool: ", out)
		if stop {
			a.Logger.Agent("agent stopping")
			if len(respLLM.Choices) > 0 {
				respLLM.Choices[0].Message.Content = out
				respLLM.Choices[0].Message.ToolCalls = nil
				respLLM.Choices[0].FinishReason = "stop"
			}
			return respLLM, nil
		}

		if err := a.CollectContext(respLLM, out); err != nil {
			a.Logger.Error("CollectContext failed", "error", err)
		}
		// log history snapshot for debugging loops
		a.Logger.Agent("History after step", "count", len(*a.History))
	}
	return models.ResponseBody{}, errors.New("limit of steps")
}

func (a *Agent) RunTool(body models.ResponseBody) (string, bool, error) {
	if len(body.Choices) == 0 || len(body.Choices[0].Message.ToolCalls) == 0 {
		return "", false, errors.New("no tool calls")
	}
	if len(body.Choices[0].Message.ToolCalls) > 1 {
		a.Logger.Warn("multiple tool calls received; only the first will be executed")
	}
	i := body.Choices[0].Message.ToolCalls[0]
	a.Logger.Agent("RunTool processing", "tool", i.Function.Name)
	switch i.Function.Name {
	case "reasoning":
		args, err := parseAgentResponse(body)
		if err != nil {
			return "", false, err
		}
		if args.Func.Function == "" {
			return "", false, nil
		}
		s, err := a.RunFunc(args)
		return s, true, err
	case "command":
		a.Logger.Agent("executing command", "args", i.Function.Arguments)
		// use command service directly for simple invocation
		svc := command.NewService()
		out, err := svc.ExecuteFromLLM(i.Function.Arguments)
		if err != nil {
			a.Logger.Warn("command execution failed", "error", err)
			return err.Error(), false, nil
		}
		return out, false, nil
	case "stop":
		var args struct {
			R string `json:"response"`
		}
		if err := json.Unmarshal([]byte(i.Function.Arguments), &args); err != nil {
			return "", false, err
		}
		*a.History = []models.Message{}
		return args.R, true, nil
	}
	return "", false, errors.New("unknown tool")
}

func (a *Agent) RunFunc(args AgentResponse) (string, error) {
	a.Logger.Agent("RunFunc called", "function", args.Func.Function)
	switch args.Func.Function {
	case "command":
		// delegate to the command service which knows how to interpret
		// the arguments string emitted by the LLM.
		svc := services.NewCommandService()
		rawArgs := string(args.Func.Arguments)
		if rawArgs == "null" {
			rawArgs = ""
		}
		out, err := svc.ExecuteFromLLM(rawArgs)
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
	if len(body.Choices) == 0 || len(body.Choices[0].Message.ToolCalls) == 0 {
		return data, errors.New("missing tool calls")
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
	tool := body.Choices[0].Message.ToolCalls[0]

	if data.Name == "reasoning" {
		args, err := parseAgentResponse(body)
		if err != nil {
			a.Logger.Error("CollectContext parseAgentResponse failed", "error", err)
			return err
		}
		if args.Func.Function != "" {
			actionArgs := string(args.Func.Arguments)
			if actionArgs == "null" {
				actionArgs = ""
			}
			content := fmt.Sprintf("args: %s\noutput: %s", actionArgs, funcOutput)
			*a.History = append(*a.History,
				models.Message{
					Role:      "function",
					ID:        tool.ID,
					Type:      tool.Type,
					Name:      args.Func.Function,
					Arguments: actionArgs,
					Output:    funcOutput,
					Content:   content,
				},
			)
		}
		// always append the assistant thought, substituting a space if it's empty
		thought := args.Thought
		if thought == "" {
			thought = " "
		}
		*a.History = append(*a.History,
			models.Message{
				Role:    "assistant",
				Content: thought,
			},
		)
		return nil
	}

	// non-reasoning tool: append tool output only
	content := fmt.Sprintf("args: %s\noutput: %s", data.Args, funcOutput)
	*a.History = append(*a.History,
		models.Message{
			Role:      "function",
			ID:        tool.ID,
			Type:      tool.Type,
			Name:      data.Name,
			Arguments: data.Args,
			Output:    funcOutput,
			Content:   content,
		},
	)

	return nil
}

// AskLLM forwards the current agent history to the LLM queue.  Before the
// request is enqueued we ensure that the optional system prompt (if set) is
// present as the first message; this lets the agent operate under a special
// instruction when running in "agent mode".
func (a *Agent) AskLLM(ToolsChoise string) (models.ResponseBody, error) {
	// insert system prompt at the beginning of the history if necessary
	if a.SystemPrompt != "" {
		if len(*a.History) == 0 || (*a.History)[0].Role != "system" || (*a.History)[0].Content != a.SystemPrompt {
			// prepend a copy so we don't mutate the original slice header
			sys := models.Message{Role: "system", Content: a.SystemPrompt}
			*a.History = append([]models.Message{sys}, *a.History...)
		}
	}

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
