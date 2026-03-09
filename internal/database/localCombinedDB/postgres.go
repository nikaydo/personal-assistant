package localcombineddb

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/nikaydo/personal-assistant/internal/models"
)

var validTableName = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type PostgresStore struct {
	mu        sync.RWMutex
	db        *sql.DB
	table     string
	dimension int
}

type Filters struct {
	IDs        []string
	Category   string
	Goal       string
	Importance string
	Status     string
	TextQuery  string
	Limit      int
	Offset     int
}

type Record struct {
	ID        string                   `json:"id"`
	Data      models.SummarizeResponse `json:"data"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
}

type vectorSearchRecord struct {
	ID       string
	Distance float32
	Data     models.SummarizeResponse
}

func NewPostgresStore(db *sql.DB, table string, dimension int) (*PostgresStore, error) {
	if db == nil {
		return nil, errors.New("postgres db is nil")
	}
	if table == "" {
		table = "summaries"
	}
	if !validTableName.MatchString(table) {
		return nil, fmt.Errorf("invalid table name: %s", table)
	}
	if dimension <= 0 {
		return nil, fmt.Errorf("invalid vector dimension: %d", dimension)
	}

	store := &PostgresStore{db: db, table: table, dimension: dimension}
	if err := store.runMigrations(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *PostgresStore) Table() string {
	return s.table
}

func (s *PostgresStore) Dimension() int {
	return s.dimension
}

func (s *PostgresStore) Upsert(id string, vector []float32, data models.SummarizeResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id == "" {
		return errors.New("id is required")
	}
	if len(vector) != s.dimension {
		return fmt.Errorf("invalid vector dimension: got %d want %d", len(vector), s.dimension)
	}

	q := fmt.Sprintf(`
INSERT INTO %s (id, category, goal, importance, status, text, embedding)
VALUES ($1, $2, $3, $4, $5, $6, $7::vector)
ON CONFLICT (id) DO UPDATE SET
	category = EXCLUDED.category,
	goal = EXCLUDED.goal,
	importance = EXCLUDED.importance,
	status = EXCLUDED.status,
	text = EXCLUDED.text,
	embedding = EXCLUDED.embedding,
	updated_at = NOW()`, s.table)

	_, err := s.db.Exec(q, id, data.Category, data.Goal, data.Importance, data.Status, data.Text, formatVector(vector))
	if err != nil {
		return fmt.Errorf("upsert postgres summary: %w", err)
	}
	return nil
}

func (s *PostgresStore) Get(id string) (Record, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if id == "" {
		return Record{}, false, errors.New("id is required")
	}

	q := fmt.Sprintf(`SELECT id, category, goal, importance, status, text, created_at, updated_at FROM %s WHERE id = $1 LIMIT 1`, s.table)
	var out Record
	err := s.db.QueryRow(q, id).Scan(
		&out.ID,
		&out.Data.Category,
		&out.Data.Goal,
		&out.Data.Importance,
		&out.Data.Status,
		&out.Data.Text,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, false, nil
		}
		return Record{}, false, fmt.Errorf("get postgres summary: %w", err)
	}
	return out, true, nil
}

func (s *PostgresStore) Search(vector []float32, topK int) ([]vectorSearchRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(vector) != s.dimension {
		return nil, fmt.Errorf("invalid vector dimension: got %d want %d", len(vector), s.dimension)
	}
	if topK <= 0 {
		topK = 10
	}

	q := fmt.Sprintf(`
SELECT id, category, goal, importance, status, text, (embedding <-> $1::vector) AS distance
FROM %s
ORDER BY embedding <-> $1::vector
LIMIT $2`, s.table)
	rows, err := s.db.Query(q, formatVector(vector), topK)
	if err != nil {
		return nil, fmt.Errorf("search postgres by vector: %w", err)
	}
	defer rows.Close()

	out := make([]vectorSearchRecord, 0, topK)
	for rows.Next() {
		var r vectorSearchRecord
		err = rows.Scan(
			&r.ID,
			&r.Data.Category,
			&r.Data.Goal,
			&r.Data.Importance,
			&r.Data.Status,
			&r.Data.Text,
			&r.Distance,
		)
		if err != nil {
			return nil, fmt.Errorf("scan postgres vector row: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postgres vector rows: %w", err)
	}
	return out, nil
}

func (s *PostgresStore) SearchByFilters(filters Filters) ([]Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	offset := max(filters.Offset, 0)

	where := make([]string, 0, 6)
	args := make([]any, 0, 12)
	argPos := 1
	if len(filters.IDs) > 0 {
		holders := make([]string, 0, len(filters.IDs))
		for _, id := range filters.IDs {
			if id == "" {
				continue
			}
			holders = append(holders, fmt.Sprintf("$%d", argPos))
			args = append(args, id)
			argPos++
		}
		if len(holders) > 0 {
			where = append(where, "id IN ("+strings.Join(holders, ",")+")")
		}
	}
	if filters.Category != "" {
		where = append(where, fmt.Sprintf("category = $%d", argPos))
		args = append(args, filters.Category)
		argPos++
	}
	if filters.Goal != "" {
		where = append(where, fmt.Sprintf("goal = $%d", argPos))
		args = append(args, filters.Goal)
		argPos++
	}
	if filters.Importance != "" {
		where = append(where, fmt.Sprintf("importance = $%d", argPos))
		args = append(args, filters.Importance)
		argPos++
	}
	if filters.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argPos))
		args = append(args, filters.Status)
		argPos++
	}
	if filters.TextQuery != "" {
		where = append(where, fmt.Sprintf("text ILIKE $%d", argPos))
		args = append(args, "%"+filters.TextQuery+"%")
		argPos++
	}

	query := fmt.Sprintf(`SELECT id, category, goal, importance, status, text, created_at, updated_at FROM %s`, s.table)
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY updated_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("search postgres summaries by filters: %w", err)
	}
	defer rows.Close()

	result := make([]Record, 0, limit)
	for rows.Next() {
		var r Record
		err = rows.Scan(
			&r.ID,
			&r.Data.Category,
			&r.Data.Goal,
			&r.Data.Importance,
			&r.Data.Status,
			&r.Data.Text,
			&r.CreatedAt,
			&r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan postgres summary row: %w", err)
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postgres summary rows: %w", err)
	}
	return result, nil
}

func formatVector(vector []float32) string {
	if len(vector) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(vector))
	for _, v := range vector {
		parts = append(parts, fmt.Sprintf("%g", v))
	}
	return "[" + strings.Join(parts, ",") + "]"
}
