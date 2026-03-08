package localcombineddb

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	hnsw "github.com/Bithack/go-hnsw"
)

const (
	defaultM              = 32
	defaultEfConstruction = 400
	defaultEfSearch       = 100
	defaultTopK           = 10
)

type HNSWRecord struct {
	InternalID uint32 `json:"internal_id"`
	ID         string `json:"id"`
	Payload    string `json:"payload,omitempty"`
	Active     bool   `json:"active"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type HNSWSearchResult struct {
	Record   HNSWRecord `json:"record"`
	Distance float32    `json:"distance"`
}

type HNSWStore struct {
	mu sync.RWMutex

	index *hnsw.Hnsw
	dim   int

	nextInternalID uint32
	externalToID   map[string]uint32
	records        map[uint32]HNSWRecord

	m              int
	efConstruction int
	efSearch       int
	defaultTopK    int

	indexFile string
	metaFile  string
}

type hnswDiskState struct {
	Dimension      int               `json:"dimension"`
	NextInternalID uint32            `json:"next_internal_id"`
	ExternalToID   map[string]uint32 `json:"external_to_id"`
	Records        []HNSWRecord      `json:"records"`
}

func NewHNSW(path string, dimension int) (*HNSWStore, error) {
	return NewHNSWWithParams(path, dimension, defaultM, defaultEfConstruction, defaultEfSearch, defaultTopK)
}

func NewHNSWWithParams(path string, dimension, m, efConstruction, efSearch, topK int) (*HNSWStore, error) {
	if dimension <= 0 {
		return nil, fmt.Errorf("invalid dimension: %d", dimension)
	}
	if m <= 0 || efConstruction <= 0 || efSearch <= 0 || topK <= 0 {
		return nil, errors.New("m, efConstruction, efSearch and topK must be > 0")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db := &HNSWStore{
		dim:            dimension,
		externalToID:   make(map[string]uint32),
		records:        make(map[uint32]HNSWRecord),
		m:              m,
		efConstruction: efConstruction,
		efSearch:       efSearch,
		defaultTopK:    topK,
		indexFile:      filepath.Join(path, "hnsw.index.gz"),
		metaFile:       filepath.Join(path, "hnsw.meta.json"),
	}
	if err := db.loadOrCreate(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *HNSWStore) Upsert(id string, vector []float32, payload string) (HNSWRecord, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if id == "" {
		return HNSWRecord{}, errors.New("id is required")
	}
	if len(vector) != db.dim {
		return HNSWRecord{}, fmt.Errorf("invalid vector dimension: got %d want %d", len(vector), db.dim)
	}

	if oldInternal, ok := db.externalToID[id]; ok {
		old := db.records[oldInternal]
		old.Active = false
		old.UpdatedAt = time.Now().Unix()
		db.records[oldInternal] = old
	}

	newInternal := db.nextInternalID + 1
	db.index.Grow(int(newInternal))
	db.index.Add(hnsw.Point(vector), newInternal)

	now := time.Now().Unix()
	rec := HNSWRecord{
		InternalID: newInternal,
		ID:         id,
		Payload:    payload,
		Active:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	db.records[newInternal] = rec
	db.externalToID[id] = newInternal
	db.nextInternalID = newInternal

	if err := db.persistLocked(); err != nil {
		return HNSWRecord{}, err
	}
	return rec, nil
}

func (db *HNSWStore) Get(id string) (HNSWRecord, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	internalID, ok := db.externalToID[id]
	if !ok {
		return HNSWRecord{}, false
	}
	rec, ok := db.records[internalID]
	if !ok || !rec.Active {
		return HNSWRecord{}, false
	}
	return rec, true
}

func (db *HNSWStore) Search(vector []float32, topK int) ([]HNSWSearchResult, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if len(vector) != db.dim {
		return nil, fmt.Errorf("invalid vector dimension: got %d want %d", len(vector), db.dim)
	}
	if topK <= 0 {
		topK = db.defaultTopK
	}
	if len(db.records) == 0 {
		return []HNSWSearchResult{}, nil
	}

	searchK := topK
	if searchK > len(db.records) {
		searchK = len(db.records)
	}

	result := db.index.Search(hnsw.Point(vector), db.efSearch, searchK*2)
	out := make([]HNSWSearchResult, 0, searchK)
	for result.Len() > 0 && len(out) < searchK {
		item := result.Pop()
		rec, ok := db.records[item.ID]
		if !ok || !rec.Active {
			continue
		}
		out = append(out, HNSWSearchResult{
			Record:   rec,
			Distance: item.D,
		})
	}
	return out, nil
}

func (db *HNSWStore) Save() error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.persistLocked()
}

func (db *HNSWStore) loadOrCreate() error {
	indexExists := fileExists(db.indexFile)
	metaExists := fileExists(db.metaFile)
	switch {
	case !indexExists && !metaExists:
		db.index = hnsw.New(db.m, db.efConstruction, make([]float32, db.dim))
		return db.persistLocked()
	case indexExists && metaExists:
		return db.loadExisting()
	default:
		return errors.New("corrupted local hnsw db: index/meta file mismatch")
	}
}

func (db *HNSWStore) loadExisting() error {
	idx, _, err := hnsw.Load(db.indexFile)
	if err != nil {
		return fmt.Errorf("load hnsw index: %w", err)
	}
	raw, err := os.ReadFile(db.metaFile)
	if err != nil {
		return fmt.Errorf("read hnsw metadata: %w", err)
	}
	var state hnswDiskState
	if err := json.Unmarshal(raw, &state); err != nil {
		return fmt.Errorf("unmarshal hnsw metadata: %w", err)
	}
	if state.Dimension != db.dim {
		return fmt.Errorf("dimension mismatch: db=%d file=%d", db.dim, state.Dimension)
	}

	db.index = idx
	db.nextInternalID = state.NextInternalID
	db.externalToID = state.ExternalToID
	db.records = make(map[uint32]HNSWRecord, len(state.Records))
	for _, rec := range state.Records {
		db.records[rec.InternalID] = rec
	}
	if db.externalToID == nil {
		db.externalToID = make(map[string]uint32)
	}
	return nil
}

func (db *HNSWStore) persistLocked() error {
	if err := db.index.Save(db.indexFile); err != nil {
		return fmt.Errorf("save hnsw index: %w", err)
	}

	state := hnswDiskState{
		Dimension:      db.dim,
		NextInternalID: db.nextInternalID,
		ExternalToID:   db.externalToID,
		Records:        make([]HNSWRecord, 0, len(db.records)),
	}
	for _, rec := range db.records {
		state.Records = append(state.Records, rec)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal hnsw metadata: %w", err)
	}
	tmp := db.metaFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write hnsw metadata temp file: %w", err)
	}
	if err := os.Rename(tmp, db.metaFile); err != nil {
		return fmt.Errorf("replace hnsw metadata: %w", err)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
