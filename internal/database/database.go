package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nikaydo/personal-assistant/internal/config"
	localCombinedDB "github.com/nikaydo/personal-assistant/internal/database/localCombinedDB"
	pn "github.com/nikaydo/personal-assistant/internal/database/pinecone"
	"github.com/nikaydo/personal-assistant/internal/models"
	"github.com/pinecone-io/go-pinecone/v5/pinecone"
)

type Pinecone = pn.DBase
type LocalDbase = localCombinedDB.DbaseVector
type Database struct {
	Secelted int

	P     Pinecone
	Local LocalDbase
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

	if cfg.LocalMySQLDSN == "" {
		return nil, errors.New("local mysql dsn is empty")
	}
	if !hasSQLDriver("mysql") {
		return nil, errors.New(`mysql driver is not registered; add: import _ "github.com/go-sql-driver/mysql"`)
	}
	if cfg.LocalVectorDim <= 0 {
		return nil, fmt.Errorf("invalid local vector dimension: %d", cfg.LocalVectorDim)
	}
	if cfg.LocalDBPath == "" {
		cfg.LocalDBPath = "localdb"
	}

	sqlDB, err := sql.Open("mysql", cfg.LocalMySQLDSN)
	if err != nil {
		return nil, fmt.Errorf("open local mysql: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping local mysql: %w", err)
	}

	local, err := localCombinedDB.Init(cfg.LocalDBPath, cfg.LocalVectorDim, sqlDB, cfg.LocalMySQLTable)
	if err != nil {
		return nil, fmt.Errorf("init local combined db: %w", err)
	}

	return &Database{
		Secelted: 2,
		Local:    *local,
	}, nil
}

func hasSQLDriver(name string) bool {
	return slices.Contains(sql.Drivers(), name)
}

func (db *Database) SaveSummary(id string, vector []float32, data models.SummarizeResponse) (localCombinedDB.SaveResult, error) {
	if db == nil {
		return localCombinedDB.SaveResult{}, errors.New("database is nil")
	}
	switch db.Secelted {
	case 2:
		return db.Local.Save(id, vector, data)
	case 1:
		if err := db.P.SaveSummary(id, vector, data); err != nil {
			return localCombinedDB.SaveResult{}, err
		}
		now := time.Now().Unix()
		payload, _ := json.Marshal(data)
		return localCombinedDB.SaveResult{
			Vector: localCombinedDB.HNSWRecord{
				ID:        id,
				Payload:   string(payload),
				Active:    true,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Data: data,
		}, nil
	default:
		return localCombinedDB.SaveResult{}, ErrUnknownDBSelection
	}
}

func (db *Database) GetSummary(id string) (localCombinedDB.FullRecord, bool, error) {
	if db == nil {
		return localCombinedDB.FullRecord{}, false, errors.New("database is nil")
	}
	switch db.Secelted {
	case 2:
		return db.Local.Get(id)
	case 1:
		data, createdAt, updatedAt, ok, err := db.P.GetSummary(id)
		if err != nil {
			return localCombinedDB.FullRecord{}, false, err
		}
		if !ok {
			return localCombinedDB.FullRecord{}, false, nil
		}
		payload, _ := json.Marshal(data)
		return localCombinedDB.FullRecord{
			ID: id,
			Vector: localCombinedDB.HNSWRecord{
				ID:        id,
				Payload:   string(payload),
				Active:    true,
				CreatedAt: createdAt.Unix(),
				UpdatedAt: updatedAt.Unix(),
			},
			Data: data,
		}, true, nil
	default:
		return localCombinedDB.FullRecord{}, false, ErrUnknownDBSelection
	}
}

func (db *Database) SearchByVector(vector []float32, topK int) ([]localCombinedDB.SearchResult, error) {
	if db == nil {
		return nil, errors.New("database is nil")
	}
	switch db.Secelted {
	case 2:
		return db.Local.Search(vector, topK)
	case 1:
		matches, err := db.P.SearchByVector(vector, topK, localCombinedDB.MySQLFilters{})
		if err != nil {
			return nil, err
		}
		out := make([]localCombinedDB.SearchResult, 0, len(matches))
		for _, m := range matches {
			out = append(out, localCombinedDB.SearchResult{
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

func (db *Database) SearchByFilters(filters localCombinedDB.MySQLFilters) ([]localCombinedDB.MySQLRecord, error) {
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
		out := make([]localCombinedDB.MySQLRecord, 0, len(matches))
		for _, m := range matches {
			out = append(out, localCombinedDB.MySQLRecord{
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
