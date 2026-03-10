package agent

import (
	"encoding/json"
	"errors"

	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/models"
)

type Agent struct {
	Steps int
	Model string
	
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

func (a *Agent) Run() {
	for range a.Steps {
		// шаг 1: вызова llm
		respLLM, err := a.AskLLM("auto")
		if err != nil {

		}
		// шаг 2: проверка на выход из агент режима
		// шаг 2: если все ок запуск функция из запроса
		_, stop, err := a.RunTool(respLLM)
		if err != nil {

		}
		if stop {
			return
		}
		// шаг 3: добавляеться в контекст размышления -> вызов функции
		a.CollectContext(respLLM, "")
		// шаг 5: повтор

	}
}

func (a *Agent) RunTool(body models.ResponseBody) (any, bool, error) {
	for _, i := range body.Choices[0].Message.ToolCalls {
		switch i.Function.Name {
		case "reasoning":
			args, err := parseAgentResponse(body)
			if err != nil {

			}
			a.RunFunc(args)
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

func (a *Agent) RunFunc(args AgentResponse) {
	switch args.Func.Function {

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

func (a *Agent) CollectContext(body models.ResponseBody, funcOutput string) {
	data, _ := getFuncNameArgs(body)
	args, _ := parseAgentResponse(body)
	tool := body.Choices[0].Message.ToolCalls[0]
	*a.History = append(*a.History,
		models.Message{
			Role:      "function",
			ID:        tool.ID,
			Type:      tool.Type,
			Name:      data.Name,
			Arguments: data.Args,
			Output:    funcOutput,
			Content:   "call and output",
		}, models.Message{
			Role:    "assistant",
			Content: args.Thought},
	)

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
