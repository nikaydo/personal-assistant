package localcombineddb

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/nikaydo/personal-assistant/internal/models"
)

func TestNewPostgresStore_Validation(t *testing.T) {
	if _, err := NewPostgresStore(nil, "summaries", 3); err == nil {
		t.Fatalf("expected nil db error")
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	if _, err := NewPostgresStore(db, "bad-table!", 3); err == nil {
		t.Fatalf("expected invalid table name error")
	}
	if _, err := NewPostgresStore(db, "summaries", 0); err == nil {
		t.Fatalf("expected invalid dimension error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected db calls: %v", err)
	}
}

func TestPostgresStore_UpsertDimensionValidation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	s := &PostgresStore{db: db, table: "summaries", dimension: 3}
	err = s.Upsert("id-1", []float32{1, 2}, models.SummarizeResponse{})
	if err == nil {
		t.Fatalf("expected dimension mismatch error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected db calls: %v", err)
	}
}

func TestPostgresStore_SearchByFilters_QueryAndScan(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "category", "goal", "importance", "status", "text", "created_at", "updated_at"}).
		AddRow("id-1", "cat", "goal", "high", "done", "txt", now, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, category, goal, importance, status, text, created_at, updated_at FROM summaries WHERE category = $1 AND text ILIKE $2 ORDER BY updated_at DESC LIMIT $3 OFFSET $4")).
		WithArgs("cat", "%needle%", 50, 0).
		WillReturnRows(rows)

	s := &PostgresStore{db: db, table: "summaries", dimension: 3}
	out, err := s.SearchByFilters(Filters{Category: "cat", TextQuery: "needle"})
	if err != nil {
		t.Fatalf("SearchByFilters error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("unexpected row count: %d", len(out))
	}
	if out[0].ID != "id-1" || out[0].Data.Text != "txt" {
		t.Fatalf("unexpected row data: %#v", out[0])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestPostgresStore_SearchByVector_QueryAndLimit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "category", "goal", "importance", "status", "text", "distance"}).
		AddRow("id-1", "cat", "goal", "high", "done", "txt", 0.12).
		AddRow("id-2", "cat", "goal", "high", "done", "txt2", 0.22)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, category, goal, importance, status, text, (embedding <-> $1::vector) AS distance\nFROM summaries\nORDER BY embedding <-> $1::vector\nLIMIT $2")).
		WithArgs("[1,2,3]", 2).
		WillReturnRows(rows)

	s := &PostgresStore{db: db, table: "summaries", dimension: 3}
	out, err := s.Search([]float32{1, 2, 3}, 2)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("unexpected count: %d", len(out))
	}
	if out[0].Distance >= out[1].Distance {
		t.Fatalf("expected ordered results by distance: %+v", out)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestFormatVector(t *testing.T) {
	got := formatVector([]float32{1, 2.5, -3})
	if got != "[1,2.5,-3]" {
		t.Fatalf("unexpected vector format: %q", got)
	}
}

func mustOpenDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
