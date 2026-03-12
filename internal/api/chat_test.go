package api

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	aimodel "github.com/nikaydo/personal-assistant/internal/ai"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
)

func TestChat_Returns501ForToolCallsNotImplemented(t *testing.T) {
	oldMakeAsk := makeAskFn
	makeAskFn = func(ai *aimodel.Ai, q string, tools []models.Tool) (models.ResponseBody, error) {
		return models.ResponseBody{}, fmt.Errorf("wrapped: %w", aimodel.ErrToolCallsNotImplemented)
	}
	t.Cleanup(func() {
		makeAskFn = oldMakeAsk
	})

	api := &API{
		Ai: &aimodel.Ai{
			Logger: &logg.Logger{
				Customlogger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"message":"hello"}`))
	rr := httptest.NewRecorder()

	api.chat(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("unexpected status code: got=%d want=%d", rr.Code, http.StatusNotImplemented)
	}
}

func TestChat_RejectsUnknownFields(t *testing.T) {
	api := &API{
		Ai: &aimodel.Ai{
			Logger: &logg.Logger{
				Customlogger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"message":"hello","extra":"x"}`))
	rr := httptest.NewRecorder()

	api.chat(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got=%d want=%d", rr.Code, http.StatusBadRequest)
	}
}

func TestChat_RejectsWhitespaceOnlyMessage(t *testing.T) {
	api := &API{
		Ai: &aimodel.Ai{
			Logger: &logg.Logger{
				Customlogger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"message":"   "}`))
	rr := httptest.NewRecorder()

	api.chat(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got=%d want=%d", rr.Code, http.StatusBadRequest)
	}
}
