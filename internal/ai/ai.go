package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nikaydo/jira-filler/internal/config"
	"github.com/nikaydo/jira-filler/internal/jira"
	"github.com/nikaydo/jira-filler/internal/logg"
	"github.com/nikaydo/jira-filler/internal/models"
)

type Ai struct {
	ApiKey    string
	Model     []string
	ModelData []models.Model

	Context Context

	Url string

	Memory *Memory

	Config config.Config

	Jira *jira.Jira

	ToolConf *ToolConf

	Logger *logg.Logger
}

type Context struct {
	ContextLeghtMax     int
	ContextLeghtCurrent int
	SummaryMemoryStep   int
}

func (ai *Ai) Ask(messages []models.Message, tools []models.Tool) (models.ResponseBody, error) {
	body := models.RequestBody{
		Model:    ai.Model[0],
		Models:   ai.Model[1:],
		Messages: messages,
	}
	if len(tools) > 0 {
		body.Tools = tools
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return models.ResponseBody{}, err
	}
	respBody, err := doReq(jsonBody, ai.Config)
	if err != nil {
		return models.ResponseBody{}, err
	}
	var response models.ResponseBody
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return models.ResponseBody{}, err
	}
	return response, nil
}

func doReq(buf []byte, config config.Config) ([]byte, error) {
	req, err := http.NewRequest("POST", config.ApiUrlOpenrouter, bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+config.ApiKeyOpenrouter)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "http://localhost")
	req.Header.Set("X-Title", "Jira filler")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

func (ai *Ai) GetModelData(config config.Config, log *logg.Logger) {
	req, err := http.NewRequest("GET", "https://openrouter.ai/api/v1/models?supported_parameters=tool", nil)
	if err != nil {
		log.Error(err)
		return
	}
	req.Header.Add("Authorization", "Bearer "+config.ApiKeyOpenrouter)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(err)
		return
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error(err)
		return
	}

	var Model models.ModelData
	err = json.Unmarshal(body, &Model)
	if err != nil {
		log.Error(err)
	}
	for _, v := range Model.Data {
		for _, i := range config.ModelOpenRouter {
			if v.Id == i {
				log.Info(fmt.Sprintf("Model %s found", v.Id))
				ai.ModelData = append(ai.ModelData, v)
				if v.ContextLength-config.HighBorderMaxContext < ai.Context.ContextLeghtMax || ai.Context.ContextLeghtMax == 0 {
					ai.Context.ContextLeghtMax = v.ContextLength - config.HighBorderMaxContext

				}
			}
		}
	}
	if config.MaxContextSize != 0 {
		ai.Context.ContextLeghtMax = config.MaxContextSize - config.HighBorderMaxContext
	}
	if len(ai.ModelData) == 0 {
		log.Error("Models not found! Need configurate settings.json file.")
		return
	}
}
