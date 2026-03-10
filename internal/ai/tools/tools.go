package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/database"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/models"
)

type Tool struct {
	Dbase *database.Database
	Queue *llmcalls.Queue
	Model string
	Cfg   config.Config
}

var createEmbeddingFn = llmcalls.CreateEmbending

func (t *Tool) DetectChosenTool(body models.ResponseBody, SystemMemory *models.SystemSettings, tools *[]models.ToolsHistory, msg []models.Message) (models.ResponseBody, error) {
	FuncName, err := GetName(body)
	if err != nil {
		return models.ResponseBody{}, err
	}
	switch FuncName {
	case "summarize":
		return models.ResponseBody{}, summarize(t, body)
	case "change_agent_settings":
		if err := change_agent_settings(body, SystemMemory); err != nil {
			return models.ResponseBody{}, err
		}
		fmt.Println(body)
		tool, msg := makeToolAndMsg(body, FuncName, msg)
		*tools = append(*tools, tool)
		var resp models.ResponseBody
		if t.Queue != nil {
			resp, err = t.AskBack(msg, []models.Tool{})
			if err != nil {
				return models.ResponseBody{}, err
			}
		}
		return resp, nil
	}
	return models.ResponseBody{}, fmt.Errorf("unknown function: %s", FuncName)
}

func summarize(t *Tool, body models.ResponseBody) error {
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
func change_agent_settings(body models.ResponseBody, SystemMemory *models.SystemSettings) error {
	var args models.SystemSettings
	if err := GetArgs(body, &args); err != nil {
		return err
	}

	dstVal := reflect.ValueOf(SystemMemory).Elem()
	srcVal := reflect.ValueOf(args)

	if dstVal.Kind() != reflect.Struct || srcVal.Kind() != reflect.Struct {
		return fmt.Errorf("both must be structs")
	}

	for i := 0; i < srcVal.NumField(); i++ {
		srcField := srcVal.Field(i)

		if srcField.Kind() == reflect.String && srcField.String() != "" {
			dstVal.Field(i).SetString(srcField.String())
		}
	}

	return nil
}
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

func (t *Tool) AskBack(msg []models.Message, tool []models.Tool) (models.ResponseBody, error) {
	respLLM, err := t.Queue.AddToQueue(llmcalls.QueueItem{Body: t.makeBody(msg, tool, "auto")})
	if err != nil {
		return models.ResponseBody{}, err
	}
	return respLLM, nil
}

func (t *Tool) makeBody(messages []models.Message, tools []models.Tool, ToolsChoise string) models.RequestBody {
	body := models.RequestBody{
		Model:       t.Model,
		Messages:    messages,
		ToolsChoise: ToolsChoise,
	}
	if len(tools) > 0 {
		body.Tools = tools
	}
	return body
}

func makeToolAndMsg(body models.ResponseBody, FuncName string, msg []models.Message) (models.ToolsHistory, []models.Message) {
	toolResult := models.ToolsHistory{
		Role:      "function",
		Type:      body.Choices[0].Message.ToolCalls[0].Type,
		Id:        body.Choices[0].Message.ToolCalls[0].ID,
		CallId:    body.Choices[0].Message.ToolCallID,
		Name:      FuncName,
		Arguments: body.Choices[0].Message.ToolCalls[0].Function.Arguments,
		Output:    body.Choices[0].Message.ToolCalls[0].Function.Arguments,
		Content:   "sucessful",
	}

	msg = append(msg, models.Message{
		Role:    toolResult.Role,
		Type:    toolResult.Type,
		ID:      toolResult.Id,
		CallId:  toolResult.CallId,
		Name:    toolResult.Name,
		Output:  toolResult.Output,
		Content: toolResult.Content,
	})
	return toolResult, msg
}
