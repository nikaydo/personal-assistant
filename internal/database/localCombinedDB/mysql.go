package localcombineddb

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

var validTableName = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type SummarizeResponse struct {
	Category   string `json:"category"`
	Goal       string `json:"goal"`
	Importance string `json:"importance"`
	Status     string `json:"status"`
	Text       string `json:"text"`
}

type MySQLStore struct {
	mu    sync.RWMutex
	db    *sql.DB
	table string
}

type MySQLFilters struct {
	IDs        []string
	Category   string
	Goal       string
	Importance string
	Status     string
	TextQuery  string
	Limit      int
	Offset     int
}

type MySQLRecord struct {
	ID        string            `json:"id"`
	Data      SummarizeResponse `json:"data"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func NewMySQLStore(db *sql.DB, table string) (*MySQLStore, error) {
	if db == nil {
		return nil, errors.New("mysql db is nil")
	}
	if table == "" {
		table = "summaries"
	}
	if !validTableName.MatchString(table) {
		return nil, fmt.Errorf("invalid table name: %s", table)
	}

	store := &MySQLStore{db: db, table: table}
	if err := store.runMigrations(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *MySQLStore) Upsert(id string, data SummarizeResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id == "" {
		return errors.New("id is required")
	}

	q := fmt.Sprintf(`
INSERT INTO %s (id, category, goal, importance, status, text)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
	category = VALUES(category),
	goal = VALUES(goal),
	importance = VALUES(importance),
	status = VALUES(status),
	text = VALUES(text),
	updated_at = CURRENT_TIMESTAMP`, s.table)

	_, err := s.db.Exec(q, id, data.Category, data.Goal, data.Importance, data.Status, data.Text)
	if err != nil {
		return fmt.Errorf("upsert mysql summary: %w", err)
	}
	return nil
}

func (s *MySQLStore) Get(id string) (SummarizeResponse, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if id == "" {
		return SummarizeResponse{}, false, errors.New("id is required")
	}

	q := fmt.Sprintf(`SELECT category, goal, importance, status, text FROM %s WHERE id = ? LIMIT 1`, s.table)
	var out SummarizeResponse
	err := s.db.QueryRow(q, id).Scan(&out.Category, &out.Goal, &out.Importance, &out.Status, &out.Text)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SummarizeResponse{}, false, nil
		}
		return SummarizeResponse{}, false, fmt.Errorf("get mysql summary: %w", err)
	}
	return out, true, nil
}

func (s *MySQLStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id == "" {
		return errors.New("id is required")
	}
	q := fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, s.table)
	_, err := s.db.Exec(q, id)
	if err != nil {
		return fmt.Errorf("delete mysql summary: %w", err)
	}
	return nil
}

func (s *MySQLStore) SearchByFilters(filters MySQLFilters) ([]MySQLRecord, error) {
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
	args := make([]any, 0, 10)
	if len(filters.IDs) > 0 {
		holders := make([]string, 0, len(filters.IDs))
		for _, id := range filters.IDs {
			if id == "" {
				continue
			}
			holders = append(holders, "?")
			args = append(args, id)
		}
		if len(holders) > 0 {
			where = append(where, "id IN ("+strings.Join(holders, ",")+")")
		}
	}
	if filters.Category != "" {
		where = append(where, "category = ?")
		args = append(args, filters.Category)
	}
	if filters.Goal != "" {
		where = append(where, "goal = ?")
		args = append(args, filters.Goal)
	}
	if filters.Importance != "" {
		where = append(where, "importance = ?")
		args = append(args, filters.Importance)
	}
	if filters.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filters.Status)
	}
	if filters.TextQuery != "" {
		where = append(where, "text LIKE ?")
		args = append(args, "%"+filters.TextQuery+"%")
	}

	query := fmt.Sprintf(`SELECT id, category, goal, importance, status, text, created_at, updated_at FROM %s`, s.table)
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY updated_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("search mysql summaries by filters: %w", err)
	}
	defer rows.Close()

	result := make([]MySQLRecord, 0, limit)
	for rows.Next() {
		var r MySQLRecord
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
			return nil, fmt.Errorf("scan mysql summary row: %w", err)
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mysql summary rows: %w", err)
	}
	return result, nil
}
