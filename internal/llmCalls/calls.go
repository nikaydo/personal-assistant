package llmcalls

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
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
	respBody, err := doReqWithRetry(jsonBody, cfg.ApiUrlOpenrouter, cfg.ApiKeyOpenrouter, "POST", cfg)
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

func CreateEmbending(input string, cfg config.Config) (models.EmbendingResponse, error) {
	body := models.EmbendingRequest{
		Model: cfg.ModelEmbending,
		Input: input,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return models.EmbendingResponse{}, err
	}
	respBody, err := doReqWithRetry(jsonBody, cfg.ApiUrlOpenrouterEmbeddings, cfg.ApiKeyOpenrouter, "POST", cfg)
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

func doReqWithRetry(buf []byte, url, token, method string, cfg config.Config) ([]byte, error) {
	attempts := cfg.LLMRetryMaxAttempts
	if attempts <= 0 {
		attempts = 3
	}
	baseDelayMs := cfg.LLMRetryBaseDelayMs
	if baseDelayMs <= 0 {
		baseDelayMs = 200
	}
	maxDelayMs := cfg.LLMRetryMaxDelayMs
	if maxDelayMs <= 0 {
		maxDelayMs = 2000
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		respBody, err := doReq(buf, url, token, method)
		if err == nil {
			return respBody, nil
		}
		lastErr = err
		if !isRetryableLLMError(err) || attempt == attempts {
			break
		}
		delay := backoffDelay(attempt, baseDelayMs, maxDelayMs)
		time.Sleep(delay)
	}
	return nil, fmt.Errorf("openrouter request failed after %d attempt(s): %w", attempts, lastErr)
}

func isRetryableLLMError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "status=408") || strings.Contains(msg, "status=409") || strings.Contains(msg, "status=425") ||
		strings.Contains(msg, "status=429") || strings.Contains(msg, "status=500") || strings.Contains(msg, "status=502") ||
		strings.Contains(msg, "status=503") || strings.Contains(msg, "status=504") {
		return true
	}
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "tempor") || strings.Contains(msg, "connection reset")
}

func backoffDelay(attempt, baseDelayMs, maxDelayMs int) time.Duration {
	mult := 1 << (attempt - 1)
	delay := baseDelayMs * mult
	if delay > maxDelayMs {
		delay = maxDelayMs
	}
	if delay <= 1 {
		return time.Duration(delay) * time.Millisecond
	}
	jitter := rand.Intn(delay / 2)
	return time.Duration(delay/2+jitter) * time.Millisecond
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
