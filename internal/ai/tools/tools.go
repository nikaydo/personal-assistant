package tools

import (
	"encoding/json"
	"errors"

	"github.com/nikaydo/personal-assistant/internal/models"
)

type Tool struct {
}

type SummarizeResponse struct {
	Category   string `json:"category"`
	Goal       string `json:"goal"`
	Importance string `json:"importance"`
	Status     string `json:"status"`
	Text       string `json:"text"`
}

func GetName(body models.ResponseBody) (string, error) {
	var FuncName string
	if len(body.Choices) == 0 {
		return "", errors.New("body not have Choices")
	}
	if len(body.Choices[0].Message.ToolCalls) == 0 {
		return "", errors.New("body not have ToolCalls")
	}
	return FuncName, json.Unmarshal([]byte(body.Choices[0].Message.ToolCalls[0].Function.Name), &FuncName)
}

func GetArgs(body models.ResponseBody, args any) error {
	if len(body.Choices) == 0 {
		return errors.New("body not have Choices")
	}
	if len(body.Choices[0].Message.ToolCalls) == 0 {
		return errors.New("body not have ToolCalls")
	}
	return json.Unmarshal([]byte(body.Choices[0].Message.ToolCalls[0].Function.Arguments), args)
}

func (t *Tool) DetectChosenTool(body models.ResponseBody) error {
	FuncName, err := GetName(body)
	if err != nil {
		return err
	}
	switch FuncName {
	case "summarize":
		var args SummarizeResponse
		if err := GetArgs(body, &args); err != nil {
			return err
		}
		//создать вектор 
		//сохранить в бд данные и сохранить в векторной бд вектор
	}
	return errors.New("unknown function")
}
