package memory

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/database"
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
	m.DBase = &database.Database{}
	m.ShortTerm = []History{
		{
			Question: ShotTermQuestion{Text: "old-q"},
			Answer:   ShotTermAnswer{Text: "old-a"},
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
	header := mem + "\n"
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
