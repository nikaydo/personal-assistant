package agent

import (
	"encoding/json"
	"errors"

	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/database"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/models"
	command "github.com/nikaydo/personal-assistant/internal/services/command"
)

type Agent struct {
	Steps int
	Model string

	Dbase *database.Database

	Cfg config.Config

	Queue *llmcalls.Queue

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

	r, err := parseAgentResponse(body)
	if err != nil {
		return models.ResponseBody{}, err
	}
	output, err := a.RunFunc(r)
	if err != nil {
		return models.ResponseBody{}, err
	}

	*a.History = append(*a.History, models.Message{
		Role:      "function",
		ID:        body.Choices[0].Message.ID,
		Type:      body.Choices[0].Message.Type,
		Name:      body.Choices[0].Message.ToolCalls[0].Function.Name,
		Arguments: body.Choices[0].Message.ToolCalls[0].Function.Arguments,
		Output:    "final",
		Content:   "call input and output"})

	if err := a.CollectContext(body, output); err != nil {
		return models.ResponseBody{}, err
	}
	for range a.Steps {
		respLLM, err := a.AskLLM("auto")
		if err != nil {
			return models.ResponseBody{}, err
		}
		out, stop, err := a.RunTool(respLLM)
		if err != nil {
			return models.ResponseBody{}, err
		}
		if stop {
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
		if err := a.CollectContext(body, out); err != nil {
			return models.ResponseBody{}, err
		}
	}
	return models.ResponseBody{}, errors.New("wtf")
}

func (a *Agent) RunTool(body models.ResponseBody) (string, bool, error) {
	for _, i := range body.Choices[0].Message.ToolCalls {
		switch i.Function.Name {
		case "reasoning":
			args, err := parseAgentResponse(body)
			if err != nil {
				return "", false, err
			}
			output, err := a.RunFunc(args)
			if err != nil {
				return "", false, err
			}
			return output, false, nil
		case "command":
			// use command service directly for simple invocation
			svc := command.NewService()
			out, err := svc.ExecuteFromLLM(i.Function.Arguments)
			if err != nil {
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
	switch args.Func.Function {
	case "command":
		// delegate to the command service which knows how to interpret
		// the arguments string emitted by the LLM.
		svc := command.NewService()
		out, err := svc.ExecuteFromLLM(args.Func.Arguments)
		if err != nil {
			return "", err
		}
		return out, nil
	default:
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
	if err := json.Unmarshal([]byte(body.Choices[0].Message.ToolCalls[0].Function.Name), &data.Name); err != nil {
		return data, err
	}
	if err := json.Unmarshal([]byte(body.Choices[0].Message.ToolCalls[0].Function.Arguments), &data.Args); err != nil {
		return data, err
	}
	data.FinishReason = body.Choices[0].FinishReason
	return data, nil
}

func (a *Agent) CollectContext(body models.ResponseBody, funcOutput string) error {
	data, err := getFuncNameArgs(body)
	if err != nil {
		return err
	}
	args, err := parseAgentResponse(body)
	if err != nil {
		return err
	}
	tool := body.Choices[0].Message.ToolCalls[0]
	*a.History = append(*a.History,
		models.Message{
			Role:      "function",
			ID:        tool.ID,
			Type:      tool.Type,
			Name:      data.Name,
			Arguments: data.Args,
			Output:    funcOutput,
			Content:   "call input and output",
		}, models.Message{
			Role:    "assistant",
			Content: args.Thought},
	)
	return nil
}

func (a *Agent) AskLLM(ToolsChoise string) (models.ResponseBody, error) {
	respLLM, err := a.Queue.AddToQueue(llmcalls.QueueItem{Body: models.RequestBody{
		Model:       a.Model,
		Messages:    *a.History,
		ToolsChoise: ToolsChoise,
	}})
	if err != nil {
		return models.ResponseBody{}, err
	}
	return respLLM, nil
}
