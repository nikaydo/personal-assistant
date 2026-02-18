package openrouter

import (
	"errors"
	"strings"

	chatmodels "github.com/nikaydo/personal-assistant/internal/models/chat"
	toolmodels "github.com/nikaydo/personal-assistant/internal/models/tool"
)

type RequestBody struct {
	Model                 string                `json:"model"`
	Models                []string              `json:"models,omitempty"`
	Messages              []chatmodels.Message  `json:"messages"`
	Provider              Provider              `json:"provider,omitempty"`
	PreferedMinThroughput PreferedMinThroughput `json:"prefered_min_throughput,omitempty"`
	Tools                 []toolmodels.Tool     `json:"tools,omitempty"`
	Input                 string                `json:"input,omitempty"`
	ToolsChoise           any                   `json:"tool_choice,omitempty"`
}

type ToolsChoise struct {
	Type     string             `json:"type"`
	Function ToolsChoisePayload `json:"function,omitempty"`
}

type ToolsChoisePayload struct {
	Name string `json:"name"`
}

type Provider struct {
	By        string `json:"by,omitempty"`
	Partition string `json:"partition,omitempty"`
}

type PreferedMinThroughput struct {
	P50 int `json:"p50,omitempty"`
	P75 int `json:"p75,omitempty"`
	P90 int `json:"p90,omitempty"`
	P99 int `json:"p99,omitempty"`
}

type ResponseBody struct {
	Error             Error     `json:"error"`
	ID                string    `json:"id"`
	Provider          string    `json:"provider"`
	Model             string    `json:"model"`
	Object            string    `json:"object"`
	Created           int64     `json:"created"`
	Choices           []Choices `json:"choices"`
	SystemFingerprint string    `json:"system_fingerprint,omitempty"`
	Usage             Usage     `json:"usage"`
}

type Error struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	Param   string `json:"param,omitempty"`
	Code    int    `json:"code,omitempty"`
}

type Choices struct {
	Logprobs           any                `json:"logprobs"`
	FinishReason       string             `json:"finish_reason"`
	NativeFinishReason string             `json:"native_finish_reason"`
	Index              int                `json:"index"`
	Message            chatmodels.Message `json:"message"`
}

type Usage struct {
	PromptTokens            int                     `json:"prompt_tokens"`
	CompletionTokens        int                     `json:"completion_tokens"`
	TotalTokens             int                     `json:"total_tokens"`
	Cost                    float64                 `json:"cost,omitempty"`
	IsByok                  bool                    `json:"is_byok,omitempty"`
	PromptTokensDetails     PromptTokensDetails     `json:"prompt_tokens_details"`
	CostDetails             CostDetails             `json:"cost_details"`
	CompletionTokensDetails CompletionTokensDetails `json:"completion_tokens_details"`
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
	AudioTokens  int `json:"audio_tokens,omitempty"`
}

type CostDetails struct {
	UpsteamInferenceCost            float64 `json:"upsteam_inference_cost,omitempty"`
	UpstreamInferencePromptCost     float64 `json:"upstream_inference_prompt_cost,omitempty"`
	UpstreamInferenceCompletionCost float64 `json:"upstream_inference_completion_cost,omitempty"`
}

type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
	AudioTokens     int `json:"audio_tokens,omitempty"`
}

func (r *ResponseBody) GetContent() string {
	if len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Message.Content
}

func ExtractJSON(s string) (string, error) {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")

	if start == -1 || end == -1 || start >= end {
		return "", errors.New("json not found")
	}
	return s[start : end+1], nil
}

type Model struct {
	Id                  string            `json:"id"`
	CanonicalSlug       string            `json:"canonical_slug"`
	Name                string            `json:"name"`
	Created             int64             `json:"created,omitempty"`
	Pricing             ModelPricing      `json:"pricing,omitempty"`
	ContextLength       int               `json:"context_length,omitempty"`
	Architecture        ModelArchitecture `json:"architecture,omitempty"`
	TopProvider         TopProvider       `json:"top_provider,omitempty"`
	PerRequestLimit     int               `json:"per_request_limit,omitempty"`
	SupportedParameters []string          `json:"supported_parameters,omitempty"`
	DefaultParameters   map[string]any    `json:"default_parameters,omitempty"`
	HuggingFaceId       string            `json:"huggingface_id,omitempty"`
	Description         string            `json:"description,omitempty"`
	ExpirationDate      string            `json:"expiration_date,omitempty"`
}

type ModelPricing struct {
	Promt             string  `json:"prompt_tokens,omitempty"`
	Completion        string  `json:"completion_tokens,omitempty"`
	Total             string  `json:"total_tokens,omitempty"`
	Request           string  `json:"request,omitempty"`
	Image             string  `json:"image,omitempty"`
	ImageToken        string  `json:"image_token,omitempty"`
	ImageOutput       string  `json:"image_output,omitempty"`
	Audio             string  `json:"audio,omitempty"`
	AudioOutput       string  `json:"audio_output,omitempty"`
	InputAudioCache   string  `json:"input_audio_cache,omitempty"`
	WebSearch         string  `json:"web_search,omitempty"`
	InternalReasoning string  `json:"internal_reasoning,omitempty"`
	InputCacheRead    string  `json:"input_cache_read,omitempty"`
	InputCacheWrite   string  `json:"input_cache_write,omitempty"`
	Discount          float64 `json:"discount,omitempty"`
}

type ModelArchitecture struct {
	Modality         string   `json:"modality,omitempty"`
	InputModalities  []string `json:"input_modalities,omitempty"`
	OutputModalities []string `json:"output_modalities,omitempty"`
	Tokenizer        string   `json:"tokenizer,omitempty"`
	InstructType     string   `json:"instruct_type,omitempty"`
}

type TopProvider struct {
	IsModerated         bool `json:"is_moderated,omitempty"`
	ContextLength       int  `json:"context_length,omitempty"`
	MaxCompletionTokens int  `json:"max_completion_tokens,omitempty"`
}

type DefaultParameters struct {
	Temp             float64 `json:"temperature,omitempty"`
	TopP             float64 `json:"top_p,omitempty"`
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"`
}

type ModelData struct {
	Data []Model `json:"data"`
}

type EmbendingResponse struct {
	Object string          `json:"object,omitempty"`
	Data   []EmbendingData `json:"data,omitempty"`
}

type EmbendingData struct {
	Object    string    `json:"object,omitempty"`
	Embedding []float32 `json:"embedding,omitempty"`
	Index     int       `json:"index,omitempty"`
	Model     string    `json:"model,omitempty"`
	Usage     Usage     `json:"usage,omitempty"`
	Provider  string    `json:"provider,omitempty"`
	Id        string    `json:"id,omitempty"`
}
