package localcombineddb

import (
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/nikaydo/personal-assistant/internal/models"
)

func TestPostgresStore_IntegrationCRUDAndVectorSearch(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not set")
	}

	db := mustOpenDB(t, dsn)
	table := fmt.Sprintf("summaries_it_%d", time.Now().UnixNano())

	s, err := NewPostgresStore(db, table, 3)
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + table)
	})

	first := models.SummarizeResponse{Category: "work", Goal: "ship", Importance: "high", Status: "done", Text: "first"}
	second := models.SummarizeResponse{Category: "work", Goal: "ship", Importance: "medium", Status: "todo", Text: "second"}

	if err := s.Upsert("id-1", []float32{0, 0, 0}, first); err != nil {
		t.Fatalf("upsert first: %v", err)
	}
	if err := s.Upsert("id-2", []float32{10, 10, 10}, second); err != nil {
		t.Fatalf("upsert second: %v", err)
	}

	rec, ok, err := s.Get("id-1")
	if err != nil || !ok {
		t.Fatalf("get id-1 failed ok=%v err=%v", ok, err)
	}
	if rec.Data.Text != "first" {
		t.Fatalf("unexpected record text: %q", rec.Data.Text)
	}

	vout, err := s.Search([]float32{0.1, 0.1, 0.1}, 2)
	if err != nil {
		t.Fatalf("vector search: %v", err)
	}
	if len(vout) != 2 {
		t.Fatalf("unexpected vector result count: %d", len(vout))
	}
	if vout[0].ID != "id-1" {
		t.Fatalf("expected id-1 to be nearest, got %s", vout[0].ID)
	}

	fout, err := s.SearchByFilters(Filters{Category: "work", Limit: 10})
	if err != nil {
		t.Fatalf("filter search: %v", err)
	}
	if len(fout) != 2 {
		t.Fatalf("unexpected filter result count: %d", len(fout))
	}
}
