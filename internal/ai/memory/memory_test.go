package memory

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func TestShouldRetrieveLongTerm_BudgetPolicy(t *testing.T) {
	m := newTestMemory()
	m.DBase = &database.Database{}
	m.Tokens.LongTermLimit = 256

	enabled, reason := m.shouldRetrieveLongTerm("какой у нас статус по задаче?", 64, DefaultBuildOptions())
	if !enabled {
		t.Fatalf("expected retrieval enabled when policy gates pass, reason=%s", reason)
	}
	if reason != "enabled_by_budget_policy" {
		t.Fatalf("unexpected reason: got=%s want=%s", reason, "enabled_by_budget_policy")
	}

	enabled, reason = m.shouldRetrieveLongTerm("какой у нас статус по задаче?", 63, DefaultBuildOptions())
	if enabled {
		t.Fatalf("expected retrieval disabled when budget is below threshold, reason=%s", reason)
	}
	if reason != "budget_below_threshold" {
		t.Fatalf("unexpected reason: got=%s want=%s", reason, "budget_below_threshold")
	}

	enabled, reason = m.shouldRetrieveLongTerm("   ", 64, DefaultBuildOptions())
	if enabled {
		t.Fatalf("expected retrieval disabled for empty question, reason=%s", reason)
	}
	if reason != "empty_question" {
		t.Fatalf("unexpected reason: got=%s want=%s", reason, "empty_question")
	}

	opts := DefaultBuildOptions()
	opts.IncludeLongTerm = false
	enabled, reason = m.shouldRetrieveLongTerm("какой у нас статус по задаче?", 64, opts)
	if enabled {
		t.Fatalf("expected retrieval disabled by options, reason=%s", reason)
	}
	if reason != "long_term_disabled_by_options" {
		t.Fatalf("unexpected reason: got=%s want=%s", reason, "long_term_disabled_by_options")
	}

	m.Tokens.LongTermLimit = 0
	enabled, reason = m.shouldRetrieveLongTerm("какой у нас статус по задаче?", 64, DefaultBuildOptions())
	if enabled {
		t.Fatalf("expected retrieval disabled when long-term limit is zero, reason=%s", reason)
	}
	if reason != "long_term_limit_disabled" {
		t.Fatalf("unexpected reason: got=%s want=%s", reason, "long_term_limit_disabled")
	}

	m.Tokens.LongTermLimit = 256
	m.DBase = nil
	enabled, reason = m.shouldRetrieveLongTerm("какой у нас статус по задаче?", 64, DefaultBuildOptions())
	if enabled {
		t.Fatalf("expected retrieval disabled when db is nil, reason=%s", reason)
	}
	if reason != "db_nil" {
		t.Fatalf("unexpected reason: got=%s want=%s", reason, "db_nil")
	}
}

func TestDynamicLongTermTopK_Capped(t *testing.T) {
	m := newTestMemory()
	k := m.dynamicLongTermTopK(strings.Repeat("a", 400), 0)
	if k > 8 {
		t.Fatalf("expected topK capped by 8, got %d", k)
	}
	if k < 2 {
		t.Fatalf("expected topK >= 2, got %d", k)
	}
}

func TestPlanContextBudget_ProducesNonNegativeBudgets(t *testing.T) {
	m := newTestMemory()
	m.Tokens.ContextLimit = 500
	m.Tokens.SystemPromptPercent = 100
	m.Tokens.SystemMemoryLimit = 50
	m.Tokens.ToolsMemoryLimit = 50
	m.Tokens.ShortTermLimit = 150
	m.Tokens.LongTermLimit = 200
	m.ShortTerm = []models.History{
		{Question: models.ShotTermQuestion{Text: "q1"}, Answer: models.ShotTermAnswer{Text: "a1"}},
	}

	plan, _ := m.PlanContextBudget("hello")
	if plan.LongTermTokenBudget < 0 {
		t.Fatalf("expected non-negative long-term budget, got %d", plan.LongTermTokenBudget)
	}
	if plan.QuestionTokens < 0 {
		t.Fatalf("expected non-negative question tokens, got %d", plan.QuestionTokens)
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
		{Question: models.ShotTermQuestion{Text: "q3"}, Answer: models.ShotTermAnswer{Text: "a3"}},
		{Question: models.ShotTermQuestion{Text: "q4"}, Answer: models.ShotTermAnswer{Text: "a4"}},
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

	if err := m.SummaryShortMemory(&llmcalls.Queue{}, "test-model"); err != nil {
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
		{Question: models.ShotTermQuestion{Text: "q3"}, Answer: models.ShotTermAnswer{Text: "a3"}},
		{Question: models.ShotTermQuestion{Text: "q4"}, Answer: models.ShotTermAnswer{Text: "a4"}},
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

	err := m.SummaryShortMemory(&llmcalls.Queue{}, "test-model")
	if err == nil {
		t.Fatalf("expected SummaryShortMemory error")
	}
	if m.Tokens.MessageCount != 4 {
		t.Fatalf("message count should stay unchanged on error: got=%d want=%d", m.Tokens.MessageCount, 4)
	}
	if len(m.ShortTerm) != 4 {
		t.Fatalf("short memory should stay unchanged on error: got=%d want=%d", len(m.ShortTerm), 4)
	}
}

func TestContextCoeffCalc_UsesConfiguredWindowAndSkipsZeroTokens(t *testing.T) {
	ct := ContextTokens{
		ContextCoeff:      []float32{1},
		ContextCoeffCount: 2,
	}

	ct.ContextCoeffCalc(160, models.ResponseBody{Usage: models.Usage{TotalTokens: 32}})
	ct.ContextCoeffCalc(320, models.ResponseBody{Usage: models.Usage{TotalTokens: 64}})
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

func TestFillShortMemory_AllowsEmptyChoices(t *testing.T) {
	m := newTestMemory()

	m.FillShortMemory("q", models.ResponseBody{
		Model: "m",
		Usage: models.Usage{TotalTokens: 3},
	})

	if len(m.ShortTerm) != 1 {
		t.Fatalf("unexpected short-term size: got=%d want=1", len(m.ShortTerm))
	}
	if m.ShortTerm[0].Answer.Text != "" {
		t.Fatalf("expected empty answer text, got %q", m.ShortTerm[0].Answer.Text)
	}
}

func TestLoadState_RecalculatesMessageCountAndSanitizesCoeff(t *testing.T) {
	m := newTestMemory()
	stateFile := filepath.Join(t.TempDir(), "memory_state.json")
	raw := `{
		"version":"v1",
		"short_term":[
			{"question":{"Text":"q1"},"answer":{"Text":"a1","Usage":{}}},
			{"question":{"Text":"q2"},"answer":{"Text":"a2","Usage":{}}}
		],
		"message_count":999,
		"context_coeff":[-1,0,4]
	}`
	if err := os.WriteFile(stateFile, []byte(raw), 0o644); err != nil {
		t.Fatalf("write state file: %v", err)
	}

	if err := m.LoadState(stateFile); err != nil {
		t.Fatalf("LoadState returned error: %v", err)
	}

	if m.Tokens.MessageCount != 2 {
		t.Fatalf("unexpected message count: got=%d want=2", m.Tokens.MessageCount)
	}
	if coeff := m.Tokens.ContextCoeffSnapshot(); len(coeff) != 1 || coeff[0] != 4 {
		t.Fatalf("unexpected coeff snapshot: %v", coeff)
	}
}

func TestSaveState_RoundTripSupportedFields(t *testing.T) {
	m := newTestMemory()
	m.SystemMemory = &models.SystemSettings{Tone: "neutral"}
	m.ToolsMemory = &[]models.ToolsHistory{{Name: "tool-a", Content: "ok"}}
	m.ShortTerm = []models.History{
		{Question: models.ShotTermQuestion{Text: "q"}, Answer: models.ShotTermAnswer{Text: "a"}},
	}
	m.Tokens.MessageCount = 7
	m.Tokens.SetContextCoeffSnapshot([]float32{2, 4})

	stateFile := filepath.Join(t.TempDir(), "memory_state.json")
	if err := m.SaveState(stateFile); err != nil {
		t.Fatalf("SaveState returned error: %v", err)
	}

	loaded := newTestMemory()
	if err := loaded.LoadState(stateFile); err != nil {
		t.Fatalf("LoadState returned error: %v", err)
	}

	if loaded.SystemMemory == nil || loaded.SystemMemory.Tone != "neutral" {
		t.Fatalf("unexpected system memory: %#v", loaded.SystemMemory)
	}
	if loaded.ToolsMemory == nil || len(*loaded.ToolsMemory) != 1 || (*loaded.ToolsMemory)[0].Name != "tool-a" {
		t.Fatalf("unexpected tools memory: %#v", loaded.ToolsMemory)
	}
	if len(loaded.ShortTerm) != 1 || loaded.ShortTerm[0].Question.Text != "q" {
		t.Fatalf("unexpected short-term: %#v", loaded.ShortTerm)
	}
	if loaded.Tokens.MessageCount != 1 {
		t.Fatalf("unexpected loaded message count: got=%d want=1", loaded.Tokens.MessageCount)
	}
	if coeff := loaded.Tokens.ContextCoeffSnapshot(); len(coeff) != 2 || coeff[0] != 2 || coeff[1] != 4 {
		t.Fatalf("unexpected loaded coeff: %v", coeff)
	}
}

func TestSummaryShortMemory_SerializesConcurrentRuns(t *testing.T) {
	m := newTestMemory()
	m.Cfg.ShortMemoryMessagesCount = 1
	m.Cfg.SummaryMemoryStep = 0
	m.Tokens.MessageCount = 2
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

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var calls atomic.Int32
	enqueueSummaryFn = func(_ *llmcalls.Queue, _ llmcalls.QueueItem) (models.ResponseBody, error) {
		calls.Add(1)
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		return models.ResponseBody{}, nil
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = m.SummaryShortMemory(&llmcalls.Queue{}, "model")
	}()
	<-started
	go func() {
		defer wg.Done()
		_ = m.SummaryShortMemory(&llmcalls.Queue{}, "model")
	}()
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("unexpected summary calls: got=%d want=1", got)
	}
	if m.Tokens.MessageCount != 1 {
		t.Fatalf("unexpected message count after summary: got=%d want=1", m.Tokens.MessageCount)
	}
}

func TestMessageWithHistory_RespectsSystemAndToolsBudget(t *testing.T) {
	m := newTestMemory()
	m.Tokens.ContextCoeff = []float32{1}
	m.Tokens.SystemPromptPercent = 100
	m.Tokens.SystemMemoryLimit = 45
	m.Tokens.ToolsMemoryLimit = 80
	m.SystemMemory = &models.SystemSettings{
		Tone:      "neutral",
		Verbosity: "detailed",
		Language:  "english",
	}
	m.ToolsMemory = &[]models.ToolsHistory{
		{Name: "tool-1", Content: "ok"},
	}

	msg := m.MessageWithHistory("q", "sys")

	if len(msg) != 3 {
		t.Fatalf("unexpected messages count: got=%d want=3", len(msg))
	}
	if msg[0].Role != "system" || !strings.Contains(msg[0].Content, "Tone: neutral") {
		t.Fatalf("unexpected system message: %#v", msg[0])
	}
	if strings.Contains(msg[0].Content, "Language: english") {
		t.Fatalf("expected personalization to be truncated by budget: %q", msg[0].Content)
	}
	if msg[1].Role != "" && msg[1].Role != "function" {
		t.Fatalf("unexpected tools memory message: %#v", msg[1])
	}
	if msg[2].Role != "user" || msg[2].Content != "q" {
		t.Fatalf("unexpected final question message: %#v", msg[2])
	}
}

func TestToolsMemoryFill_RespectsBudget(t *testing.T) {
	m := newTestMemory()
	m.Tokens.ContextCoeff = []float32{1}
	m.Tokens.ToolsMemoryLimit = 70
	m.ToolsMemory = &[]models.ToolsHistory{
		{Name: "tool-1", Content: strings.Repeat("a", 40)},
		{Name: "tool-2", Content: strings.Repeat("b", 40)},
	}

	toolTokens := 0
	msg := m.ToolsMemoryFill(nil, &toolTokens)

	if len(msg) != 1 {
		t.Fatalf("unexpected tools messages count: got=%d want=1", len(msg))
	}
	if !strings.Contains(msg[0].Content, strings.Repeat("a", 40)) {
		t.Fatalf("unexpected tool content: %#v", msg[0])
	}
}
