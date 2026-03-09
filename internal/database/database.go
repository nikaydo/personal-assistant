package database

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/nikaydo/personal-assistant/internal/config"
	localCombinedDB "github.com/nikaydo/personal-assistant/internal/database/localCombinedDB"
	pn "github.com/nikaydo/personal-assistant/internal/database/pinecone"
	"github.com/nikaydo/personal-assistant/internal/models"
	"github.com/pinecone-io/go-pinecone/v5/pinecone"
)

type Pinecone = pn.DBase
type LocalDbase = localCombinedDB.DbaseVector
type Filters = localCombinedDB.Filters
type Record = localCombinedDB.Record
type SearchResult = localCombinedDB.SearchResult
type SaveResult = localCombinedDB.SaveResult
type FullRecord = localCombinedDB.FullRecord

type Database struct {
	Secelted int

	P     Pinecone
	Local LocalDbase
	sqlDB *sql.DB
}

var (
	ErrUnknownDBSelection   = errors.New("unknown db selection")
	ErrOperationUnsupported = errors.New("operation is not supported for selected database")
)

func InitDB(cfg *config.Config) (*Database, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	if cfg.PinecoreApiKey != "" {
		pc, err := pinecone.NewClient(pinecone.NewClientParams{
			ApiKey: cfg.PinecoreApiKey,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create pinecone client: %w", err)
		}
		pdb := Pinecone{
			Client: pc,
			Cfg:    cfg,
		}
		if err := pdb.EnsureReady(); err != nil {
			return nil, fmt.Errorf("ensure pinecone index ready: %w", err)
		}
		return &Database{
			Secelted: 1,
			P:        pdb,
		}, nil
	}

	if cfg.LocalPostgresDSN == "" {
		return nil, errors.New("local postgres dsn is empty")
	}
	if !hasSQLDriver("pgx") {
		return nil, errors.New(`postgres driver is not registered; add: import _ "github.com/jackc/pgx/v5/stdlib"`)
	}
	if cfg.LocalVectorDim <= 0 {
		return nil, fmt.Errorf("invalid local vector dimension: %d", cfg.LocalVectorDim)
	}

	sqlDB, err := sql.Open("pgx", cfg.LocalPostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("open local postgres: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping local postgres: %w", err)
	}

	local, err := localCombinedDB.Init(cfg.LocalVectorDim, sqlDB, cfg.LocalPostgresTable)
	if err != nil {
		return nil, fmt.Errorf("init local postgres db: %w", err)
	}

	return &Database{
		Secelted: 2,
		Local:    *local,
		sqlDB:    sqlDB,
	}, nil
}

func hasSQLDriver(name string) bool {
	return slices.Contains(sql.Drivers(), name)
}

func (db *Database) SaveSummary(id string, vector []float32, data models.SummarizeResponse) (SaveResult, error) {
	if db == nil {
		return SaveResult{}, errors.New("database is nil")
	}
	switch db.Secelted {
	case 2:
		return db.Local.Save(id, vector, data)
	case 1:
		if err := db.P.SaveSummary(id, vector, data); err != nil {
			return SaveResult{}, err
		}
		return SaveResult{
			Vector: localCombinedDB.VectorRecord{
				ID:        id,
				Dimension: len(vector),
				Active:    true,
			},
			Data: data,
		}, nil
	default:
		return SaveResult{}, ErrUnknownDBSelection
	}
}

func (db *Database) GetSummary(id string) (FullRecord, bool, error) {
	if db == nil {
		return FullRecord{}, false, errors.New("database is nil")
	}
	switch db.Secelted {
	case 2:
		return db.Local.Get(id)
	case 1:
		data, createdAt, updatedAt, ok, err := db.P.GetSummary(id)
		if err != nil {
			return FullRecord{}, false, err
		}
		if !ok {
			return FullRecord{}, false, nil
		}
		_ = createdAt
		_ = updatedAt
		return FullRecord{
			ID: id,
			Vector: localCombinedDB.VectorRecord{
				ID:        id,
				Dimension: 0,
				Active:    true,
			},
			Data: data,
		}, true, nil
	default:
		return FullRecord{}, false, ErrUnknownDBSelection
	}
}

func (db *Database) SearchByVector(vector []float32, topK int) ([]SearchResult, error) {
	if db == nil {
		return nil, errors.New("database is nil")
	}
	switch db.Secelted {
	case 2:
		return db.Local.Search(vector, topK)
	case 1:
		matches, err := db.P.SearchByVector(vector, topK, Filters{})
		if err != nil {
			return nil, err
		}
		out := make([]SearchResult, 0, len(matches))
		for _, m := range matches {
			out = append(out, SearchResult{
				ID:       m.ID,
				Distance: m.Score,
				Data:     m.Data,
			})
		}
		return out, nil
	default:
		return nil, ErrUnknownDBSelection
	}
}

func (db *Database) SearchByFilters(filters Filters) ([]Record, error) {
	if db == nil {
		return nil, errors.New("database is nil")
	}
	switch db.Secelted {
	case 2:
		return db.Local.SearchByFilters(filters)
	case 1:
		matches, err := db.P.SearchByFilters(filters)
		if err != nil {
			return nil, err
		}
		out := make([]Record, 0, len(matches))
		for _, m := range matches {
			out = append(out, Record{
				ID:        m.ID,
				Data:      m.Data,
				CreatedAt: m.CreatedAt,
				UpdatedAt: m.UpdatedAt,
			})
		}
		return out, nil
	default:
		return nil, ErrUnknownDBSelection
	}
}

func (db *Database) UpsertPineconeRecords(records []*pinecone.IntegratedRecord) error {
	if db == nil {
		return errors.New("database is nil")
	}
	switch db.Secelted {
	case 1:
		return db.P.Upsert(records)
	case 2:
		return fmt.Errorf("%w: UpsertPineconeRecords for local db", ErrOperationUnsupported)
	default:
		return ErrUnknownDBSelection
	}
}

func (db *Database) Close() error {
	if db == nil || db.sqlDB == nil {
		return nil
	}
	return db.sqlDB.Close()
}
