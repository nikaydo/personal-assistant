package llmcalls

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func Ask(body models.RequestBody, cfg config.Config) (models.ResponseBody, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return models.ResponseBody{}, err
	}
	respBody, err := doReq(jsonBody, cfg.ApiUrlOpenrouter, cfg.ApiKeyOpenrouter, "POST")
	if err != nil {
		// wrap error with request body so callers (and logs) can inspect what was sent
		return models.ResponseBody{}, fmt.Errorf("%w request=%s", err, string(jsonBody))
	}
	var response models.ResponseBody
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return models.ResponseBody{}, err
	}
	return response, nil
}

func CreateEmbending(input string, cfg config.Config) (models.EmbendingResponse, error) {
	body := models.EmbendingRequest{
		Model: cfg.ModelEmbending,
		Input: input,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return models.EmbendingResponse{}, err
	}
	respBody, err := doReq(jsonBody, cfg.ApiUrlOpenrouterEmbeddings, cfg.ApiKeyOpenrouter, "POST")
	if err != nil {
		return models.EmbendingResponse{}, err
	}
	var response models.EmbendingResponse
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return models.EmbendingResponse{}, err
	}
	return response, nil
}

func doReq(buf []byte, url, token, method string) ([]byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "http://localhost")
	req.Header.Set("X-Title", "narria")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openrouter request failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

func GetModelData(cfg config.Config, log *logg.Logger) (models.ModelData, error) {
	req, err := doReq([]byte{}, "https://openrouter.ai/api/v1/models?supported_parameters=tool,tool_choice", cfg.ApiKeyOpenrouter, "GET")
	if err != nil {
		log.Error(err)
		return models.ModelData{}, err
	}
	var Models models.ModelData
	err = json.Unmarshal(req, &Models)
	if err != nil {
		log.Error(err)
		return models.ModelData{}, err
	}
	return Models, nil
}
