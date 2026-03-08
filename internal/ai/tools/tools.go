package tools

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/database"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/models"
)

type Tool struct {
	Dbase *database.Database
	Cfg   config.Config
}

var createEmbeddingFn = llmcalls.CreateEmbending

func GetName(body models.ResponseBody) (string, error) {
	if len(body.Choices) == 0 {
		return "", errors.New("body not have Choices")
	}
	if len(body.Choices[0].Message.ToolCalls) == 0 {
		return "", errors.New("body not have ToolCalls")
	}
	return body.Choices[0].Message.ToolCalls[0].Function.Name, nil
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
		var args models.SummarizeResponse
		if err := GetArgs(body, &args); err != nil {
			return err
		}
		emb, err := createEmbeddingFn(args.Text, t.Cfg)
		if err != nil {
			return err
		}
		if len(emb.Data) == 0 || len(emb.Data[0].Embedding) == 0 {
			return errors.New("empty embedding response")
		}
		if t.Dbase == nil {
			return errors.New("database is nil")
		}
		if _, err := t.Dbase.SaveSummary(uuid.New().String(), emb.Data[0].Embedding, args); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("unknown function: %s", FuncName)
}
