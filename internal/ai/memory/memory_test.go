package memory

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	toolsmemory "github.com/nikaydo/personal-assistant/internal/ai/tools"
	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/database"
	llmcalls "github.com/nikaydo/personal-assistant/internal/llmCalls"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
)

func newTestLogger() *logg.Logger {
	return &logg.Logger{
		Customlogger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func newTestMemory() *Memory {
	return &Memory{
		Logger: newTestLogger(),
		Cfg:    config.Config{},
		Tokens: ContextTokens{
			ContextCoeff: []float32{1},
		},
	}
}

func withEmbeddingSearchStubs(t *testing.T, embFn func(string, config.Config) (models.EmbendingResponse, error), searchFn func(*database.Database, []float32, int) ([]models.SummarizeResponse, error)) {
	t.Helper()
	oldEmb := createEmbeddingFn
	oldSearch := searchByVectorFn
	createEmbeddingFn = embFn
	searchByVectorFn = searchFn
	t.Cleanup(func() {
		createEmbeddingFn = oldEmb
		searchByVectorFn = oldSearch
	})
}

func embeddingResponse(t *testing.T, vector []float32) models.EmbendingResponse {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"data": []map[string]any{
			{
				"embedding": vector,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal embedding response: %v", err)
	}
	var out models.EmbendingResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal embedding response: %v", err)
	}
	return out
}

func TestLongTermMemoryFill_SkipWhenDisabled(t *testing.T) {
	m := newTestMemory()
	m.Tokens.LongTermLimit = 0

	embeddingCalled := false
	withEmbeddingSearchStubs(t,
		func(input string, cfg config.Config) (models.EmbendingResponse, error) {
			embeddingCalled = true
			return models.EmbendingResponse{}, nil
		},
		func(*database.Database, []float32, int) ([]models.SummarizeResponse, error) {
			return nil, nil
		},
	)

	tokens := 0
	msg := m.LongTermMemoryFill("hello", []models.Message{}, &tokens)

	if embeddingCalled {
		t.Fatalf("embedding should not be called when long-term memory is disabled")
	}
	if len(msg) != 0 {
		t.Fatalf("expected no message appended, got %d", len(msg))
	}
	if tokens != 0 {
		t.Fatalf("expected 0 tokens, got %d", tokens)
	}
}

func TestLongTermMemoryFill_SkipWhenQuestionEmpty(t *testing.T) {
	m := newTestMemory()
	m.Tokens.LongTermLimit = 100

	embeddingCalled := false
	withEmbeddingSearchStubs(t,
		func(input string, cfg config.Config) (models.EmbendingResponse, error) {
			embeddingCalled = true
			return models.EmbendingResponse{}, nil
		},
		func(*database.Database, []float32, int) ([]models.SummarizeResponse, error) {
			return nil, nil
		},
	)

	tokens := 0
	msg := m.LongTermMemoryFill("", []models.Message{}, &tokens)

	if embeddingCalled {
		t.Fatalf("embedding should not be called when question is empty")
	}
	if len(msg) != 0 {
		t.Fatalf("expected no message appended, got %d", len(msg))
	}
}

func TestMessageWithHistory_LongTermBlockInjectedInOrder(t *testing.T) {
	m := newTestMemory()
	m.Tokens.LongTermLimit = 10_000
	m.Tokens.ShortTermLimit = 10_000
	m.DBase = &database.Database{}
	m.ShortTerm = []models.History{
		{
			Question: models.ShotTermQuestion{Text: "old-q"},
			Answer:   models.ShotTermAnswer{Text: "old-a"},
		},
	}

	withEmbeddingSearchStubs(t,
		func(input string, cfg config.Config) (models.EmbendingResponse, error) {
			return embeddingResponse(t, []float32{1, 2}), nil
		},
		func(*database.Database, []float32, int) ([]models.SummarizeResponse, error) {
			return []models.SummarizeResponse{
				{Text: "first fact"},
				{Text: "second fact"},
			}, nil
		},
	)

	msg := m.MessageWithHistory("new-q", "")
	if len(msg) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msg))
	}
	if msg[0].Role != "system" {
		t.Fatalf("expected first message to be system, got %q", msg[0].Role)
	}
	if !strings.Contains(msg[0].Content, "1. first fact") || !strings.Contains(msg[0].Content, "2. second fact") {
		t.Fatalf("unexpected long-term content: %q", msg[0].Content)
	}
	if strings.Index(msg[0].Content, "1. first fact") > strings.Index(msg[0].Content, "2. second fact") {
		t.Fatalf("expected backend order to be preserved, got: %q", msg[0].Content)
	}
	if msg[1].Role != "user" || msg[1].Content != "old-q" {
		t.Fatalf("unexpected short-term user message: %#v", msg[1])
	}
	if msg[2].Role != "assistant" || msg[2].Content != "old-a" {
		t.Fatalf("unexpected short-term assistant message: %#v", msg[2])
	}
	if msg[3].Role != "user" || msg[3].Content != "new-q" {
		t.Fatalf("unexpected final question message: %#v", msg[3])
	}
}

func TestBuildLongTermBlock_RespectsTokenBudget(t *testing.T) {
	m := newTestMemory()
	header := "Long-term conversation memory:\n"
	firstLine := "1. one\n"
	m.Tokens.LongTermLimit = len(header) + len(firstLine)

	content, tokens, _ := m.buildLongTermBlock([]models.SummarizeResponse{
		{Text: "one"},
		{Text: "two"},
	})

	if content == "" {
		t.Fatalf("expected non-empty long-term block")
	}
	if !strings.Contains(content, "1. one") {
		t.Fatalf("expected first entry in block, got %q", content)
	}
	if strings.Contains(content, "2. two") {
		t.Fatalf("did not expect second entry due to budget limit, got %q", content)
	}
	if tokens > m.Tokens.LongTermLimit {
		t.Fatalf("tokens exceeded limit: got=%d limit=%d", tokens, m.Tokens.LongTermLimit)
	}
}

func TestMessageWithHistory_LongTermSoftFallbackOnErrors(t *testing.T) {
	t.Run("embedding error", func(t *testing.T) {
		m := newTestMemory()
		m.Tokens.LongTermLimit = 10_000
		m.DBase = &database.Database{}

		withEmbeddingSearchStubs(t,
			func(input string, cfg config.Config) (models.EmbendingResponse, error) {
				return models.EmbendingResponse{}, errors.New("embedding failed")
			},
			func(*database.Database, []float32, int) ([]models.SummarizeResponse, error) {
				return []models.SummarizeResponse{{Text: "should not be used"}}, nil
			},
		)

		msg := m.MessageWithHistory("new-q", "")
		if len(msg) != 1 {
			t.Fatalf("expected only user question after fallback, got %d messages", len(msg))
		}
		if msg[0].Role != "user" || msg[0].Content != "new-q" {
			t.Fatalf("unexpected fallback message: %#v", msg[0])
		}
	})

	t.Run("search error", func(t *testing.T) {
		m := newTestMemory()
		m.Tokens.LongTermLimit = 10_000
		m.DBase = &database.Database{}

		withEmbeddingSearchStubs(t,
			func(input string, cfg config.Config) (models.EmbendingResponse, error) {
				return embeddingResponse(t, []float32{1}), nil
			},
			func(*database.Database, []float32, int) ([]models.SummarizeResponse, error) {
				return nil, errors.New("search failed")
			},
		)

		msg := m.MessageWithHistory("new-q", "")
		if len(msg) != 1 {
			t.Fatalf("expected only user question after fallback, got %d messages", len(msg))
		}
		if msg[0].Role != "user" || msg[0].Content != "new-q" {
			t.Fatalf("unexpected fallback message: %#v", msg[0])
		}
	})
}

func TestSummaryShortMemory_TriggersWhenMessageCountAboveThreshold(t *testing.T) {
	m := newTestMemory()
	m.Cfg.ShortMemoryMessagesCount = 2
	m.Cfg.SummaryMemoryStep = 1
	m.Tokens.MessageCount = 4
	m.ShortTerm = []models.History{
		{Question: models.ShotTermQuestion{Text: "q1"}, Answer: models.ShotTermAnswer{Text: "a1"}},
		{Question: models.ShotTermQuestion{Text: "q2"}, Answer: models.ShotTermAnswer{Text: "a2"}},
	}

	oldEnqueue := enqueueSummaryFn
	oldDetect := detectChosenToolFn
	t.Cleanup(func() {
		enqueueSummaryFn = oldEnqueue
		detectChosenToolFn = oldDetect
	})

	enqueueCalled := false
	enqueueSummaryFn = func(_ *llmcalls.Queue, _ llmcalls.QueueItem) (models.ResponseBody, error) {
		enqueueCalled = true
		return models.ResponseBody{}, nil
	}
	detectChosenToolFn = func(_ *toolsmemory.Tool, _ models.ResponseBody) error {
		return nil
	}

	if err := m.SummaryShortMemory(nil, "test-model"); err != nil {
		t.Fatalf("SummaryShortMemory returned error: %v", err)
	}
	if !enqueueCalled {
		t.Fatalf("expected summarization queue call when message count is above threshold")
	}
	if m.Tokens.MessageCount != m.Cfg.ShortMemoryMessagesCount {
		t.Fatalf("unexpected message count after summarization: got=%d want=%d", m.Tokens.MessageCount, m.Cfg.ShortMemoryMessagesCount)
	}
}

func TestSummaryShortMemory_DoesNotDropMessagesOnEnqueueError(t *testing.T) {
	m := newTestMemory()
	m.Cfg.ShortMemoryMessagesCount = 2
	m.Cfg.SummaryMemoryStep = 1
	m.Tokens.MessageCount = 4
	m.ShortTerm = []models.History{
		{Question: models.ShotTermQuestion{Text: "q1"}, Answer: models.ShotTermAnswer{Text: "a1"}},
		{Question: models.ShotTermQuestion{Text: "q2"}, Answer: models.ShotTermAnswer{Text: "a2"}},
	}

	oldEnqueue := enqueueSummaryFn
	oldDetect := detectChosenToolFn
	t.Cleanup(func() {
		enqueueSummaryFn = oldEnqueue
		detectChosenToolFn = oldDetect
	})

	enqueueSummaryFn = func(_ *llmcalls.Queue, _ llmcalls.QueueItem) (models.ResponseBody, error) {
		return models.ResponseBody{}, errors.New("queue failed")
	}
	detectChosenToolFn = func(_ *toolsmemory.Tool, _ models.ResponseBody) error {
		return nil
	}

	err := m.SummaryShortMemory(nil, "test-model")
	if err == nil {
		t.Fatalf("expected SummaryShortMemory error")
	}
	if m.Tokens.MessageCount != 4 {
		t.Fatalf("message count should stay unchanged on error: got=%d want=%d", m.Tokens.MessageCount, 4)
	}
	if len(m.ShortTerm) != 2 {
		t.Fatalf("short memory should stay unchanged on error: got=%d want=%d", len(m.ShortTerm), 2)
	}
}

func TestSummaryShortMemory_UsesConfigPrompt(t *testing.T) {
	m := newTestMemory()
	m.Cfg.ShortMemoryMessagesCount = 1
	m.Cfg.SummaryMemoryStep = 1
	m.Cfg.PromtMemorySummary = "prompt from config"
	m.Tokens.MessageCount = 2
	m.ShortTerm = []models.History{
		{Question: models.ShotTermQuestion{Text: "q1"}, Answer: models.ShotTermAnswer{Text: "a1"}},
	}

	oldEnqueue := enqueueSummaryFn
	oldDetect := detectChosenToolFn
	t.Cleanup(func() {
		enqueueSummaryFn = oldEnqueue
		detectChosenToolFn = oldDetect
	})

	var gotPrompt string
	enqueueSummaryFn = func(_ *llmcalls.Queue, item llmcalls.QueueItem) (models.ResponseBody, error) {
		if len(item.Body.Messages) > 0 {
			gotPrompt = item.Body.Messages[0].Content
		}
		return models.ResponseBody{}, nil
	}
	detectChosenToolFn = func(_ *toolsmemory.Tool, _ models.ResponseBody) error {
		return nil
	}

	if err := m.SummaryShortMemory(nil, "test-model"); err != nil {
		t.Fatalf("SummaryShortMemory returned error: %v", err)
	}
	if gotPrompt != m.Cfg.PromtMemorySummary {
		t.Fatalf("unexpected summary prompt: got=%q want=%q", gotPrompt, m.Cfg.PromtMemorySummary)
	}
}

func TestContextCoeffCalc_UsesConfiguredWindowAndSkipsZeroTokens(t *testing.T) {
	ct := ContextTokens{
		ContextCoeff:      []float32{1},
		ContextCoeffCount: 2,
	}

	ct.ContextCoeffCalc(10, models.ResponseBody{Usage: models.Usage{TotalTokens: 2}})
	ct.ContextCoeffCalc(20, models.ResponseBody{Usage: models.Usage{TotalTokens: 4}})
	ct.ContextCoeffCalc(999, models.ResponseBody{Usage: models.Usage{TotalTokens: 0}})

	coeffs := ct.ContextCoeffSnapshot()
	if len(coeffs) != 2 {
		t.Fatalf("unexpected coeff window size: got=%d want=2", len(coeffs))
	}
	if coeffs[0] != 5 || coeffs[1] != 5 {
		t.Fatalf("unexpected coeffs: got=%v want=[5 5]", coeffs)
	}
}

func TestShortMemoryFill_RespectsTokenLimit(t *testing.T) {
	m := newTestMemory()
	m.Tokens.ContextCoeff = []float32{1}
	m.Tokens.ShortTermLimit = 6
	m.ShortTerm = []models.History{
		{Question: models.ShotTermQuestion{Text: "a"}, Answer: models.ShotTermAnswer{Text: "b"}},
		{Question: models.ShotTermQuestion{Text: "cc"}, Answer: models.ShotTermAnswer{Text: "dd"}},
		{Question: models.ShotTermQuestion{Text: "eee"}, Answer: models.ShotTermAnswer{Text: "fff"}},
	}

	shortTokens := 0
	msg := m.ShortMemoryFill(nil, &shortTokens)

	if len(msg) != 2 {
		t.Fatalf("unexpected messages count: got=%d want=2", len(msg))
	}
	if msg[0].Role != "user" || msg[0].Content != "eee" {
		t.Fatalf("unexpected user message: %#v", msg[0])
	}
	if msg[1].Role != "assistant" || msg[1].Content != "fff" {
		t.Fatalf("unexpected assistant message: %#v", msg[1])
	}
	if shortTokens != 6 {
		t.Fatalf("unexpected short-term tokens: got=%d want=6", shortTokens)
	}
}
