package localcombineddb

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/nikaydo/personal-assistant/internal/models"
)

type Store struct {
	PG *PostgresStore
}

// Backward-compatible alias used by internal/database/database.go.
type DbaseVector = Store

type VectorRecord struct {
	ID        string `json:"id"`
	Dimension int    `json:"dimension"`
	Active    bool   `json:"active"`
}

type SaveResult struct {
	Vector VectorRecord             `json:"vector"`
	Data   models.SummarizeResponse `json:"data"`
}

type FullRecord struct {
	ID     string                   `json:"id"`
	Vector VectorRecord             `json:"vector"`
	Data   models.SummarizeResponse `json:"data"`
}

type SearchResult struct {
	ID       string                   `json:"id"`
	Distance float32                  `json:"distance"`
	Data     models.SummarizeResponse `json:"data"`
}

func NewCombined(pgStore *PostgresStore) (*Store, error) {
	if pgStore == nil {
		return nil, errors.New("postgres store is nil")
	}
	return &Store{PG: pgStore}, nil
}

func Init(dimension int, sqlDB *sql.DB, table string) (*Store, error) {
	pgStore, err := NewPostgresStore(sqlDB, table, dimension)
	if err != nil {
		return nil, err
	}
	return NewCombined(pgStore)
}

func (db *Store) Save(id string, vector []float32, data models.SummarizeResponse) (SaveResult, error) {
	if db == nil || db.PG == nil {
		return SaveResult{}, errors.New("local db is not initialized")
	}

	if err := db.PG.Upsert(id, vector, data); err != nil {
		return SaveResult{}, err
	}

	return SaveResult{
		Vector: VectorRecord{ID: id, Dimension: len(vector), Active: true},
		Data:   data,
	}, nil
}

func (db *Store) Get(id string) (FullRecord, bool, error) {
	if db == nil || db.PG == nil {
		return FullRecord{}, false, errors.New("local db is not initialized")
	}

	rec, ok, err := db.PG.Get(id)
	if err != nil {
		return FullRecord{}, false, err
	}
	if !ok {
		return FullRecord{}, false, nil
	}

	return FullRecord{
		ID: id,
		Vector: VectorRecord{
			ID:        id,
			Dimension: db.PG.Dimension(),
			Active:    true,
		},
		Data: rec.Data,
	}, true, nil
}

func (db *Store) Search(vector []float32, topK int) ([]SearchResult, error) {
	if db == nil || db.PG == nil {
		return nil, errors.New("local db is not initialized")
	}

	records, err := db.PG.Search(vector, topK)
	if err != nil {
		return nil, err
	}

	out := make([]SearchResult, 0, len(records))
	for _, rec := range records {
		out = append(out, SearchResult{
			ID:       rec.ID,
			Distance: rec.Distance,
			Data:     rec.Data,
		})
	}
	return out, nil
}

func (db *Store) SearchByFilters(filters Filters) ([]Record, error) {
	if db == nil || db.PG == nil {
		return nil, errors.New("local db is not initialized")
	}
	return db.PG.SearchByFilters(filters)
}

func (db *Store) DebugString() string {
	if db == nil || db.PG == nil {
		return "local store: nil"
	}
	return fmt.Sprintf("local store: postgres table=%s dim=%d", db.PG.Table(), db.PG.Dimension())
}
