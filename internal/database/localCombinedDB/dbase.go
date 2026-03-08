package localcombineddb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type Dbase struct {
	HNSW  *HNSWStore
	MySQL *MySQLStore
}

// Backward-compatible alias used by internal/database/dbase.go.
type DbaseVector = Dbase

type SaveResult struct {
	Vector HNSWRecord        `json:"vector"`
	Data   SummarizeResponse `json:"data"`
}

type FullRecord struct {
	ID     string            `json:"id"`
	Vector HNSWRecord        `json:"vector"`
	Data   SummarizeResponse `json:"data"`
}

type SearchResult struct {
	ID       string            `json:"id"`
	Distance float32           `json:"distance"`
	Data     SummarizeResponse `json:"data"`
}

func NewCombined(hnswStore *HNSWStore, mysqlStore *MySQLStore) (*Dbase, error) {
	if hnswStore == nil {
		return nil, errors.New("hnsw store is nil")
	}
	if mysqlStore == nil {
		return nil, errors.New("mysql store is nil")
	}
	return &Dbase{HNSW: hnswStore, MySQL: mysqlStore}, nil
}

func Init(path string, dimension int, sqlDB *sql.DB, mysqlTable string) (*Dbase, error) {
	hnswStore, err := NewHNSW(path, dimension)
	if err != nil {
		return nil, err
	}
	mysqlStore, err := NewMySQLStore(sqlDB, mysqlTable)
	if err != nil {
		return nil, err
	}
	return NewCombined(hnswStore, mysqlStore)
}

func (db *Dbase) Save(id string, vector []float32, data SummarizeResponse) (SaveResult, error) {
	if db == nil || db.HNSW == nil || db.MySQL == nil {
		return SaveResult{}, errors.New("combined db is not initialized")
	}
	if id == "" {
		return SaveResult{}, errors.New("id is required")
	}

	if err := db.MySQL.Upsert(id, data); err != nil {
		return SaveResult{}, err
	}

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return SaveResult{}, fmt.Errorf("marshal summary payload: %w", err)
	}

	vecRec, err := db.HNSW.Upsert(id, vector, string(payloadBytes))
	if err != nil {
		rollbackErr := db.MySQL.Delete(id)
		if rollbackErr != nil {
			return SaveResult{}, fmt.Errorf("hnsw save failed: %v; mysql rollback failed: %w", err, rollbackErr)
		}
		return SaveResult{}, err
	}

	return SaveResult{Vector: vecRec, Data: data}, nil
}

func (db *Dbase) Get(id string) (FullRecord, bool, error) {
	if db == nil || db.HNSW == nil || db.MySQL == nil {
		return FullRecord{}, false, errors.New("combined db is not initialized")
	}

	vecRec, ok := db.HNSW.Get(id)
	if !ok {
		return FullRecord{}, false, nil
	}
	data, ok, err := db.MySQL.Get(id)
	if err != nil {
		return FullRecord{}, false, err
	}
	if !ok {
		return FullRecord{}, false, nil
	}

	return FullRecord{
		ID:     id,
		Vector: vecRec,
		Data:   data,
	}, true, nil
}

func (db *Dbase) Search(vector []float32, topK int) ([]SearchResult, error) {
	if db == nil || db.HNSW == nil || db.MySQL == nil {
		return nil, errors.New("combined db is not initialized")
	}

	candidates, err := db.HNSW.Search(vector, topK)
	if err != nil {
		return nil, err
	}

	out := make([]SearchResult, 0, len(candidates))
	for _, c := range candidates {
		data, ok, err := db.MySQL.Get(c.Record.ID)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		out = append(out, SearchResult{
			ID:       c.Record.ID,
			Distance: c.Distance,
			Data:     data,
		})
	}
	return out, nil
}

func (db *Dbase) SearchByFilters(filters MySQLFilters) ([]MySQLRecord, error) {
	if db == nil || db.MySQL == nil {
		return nil, errors.New("combined db mysql is not initialized")
	}
	return db.MySQL.SearchByFilters(filters)
}
