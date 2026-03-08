package database

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nikaydo/personal-assistant/internal/config"
	localCombinedDB "github.com/nikaydo/personal-assistant/internal/database/localCombinedDB"
	pn "github.com/nikaydo/personal-assistant/internal/database/pinecone"
	"github.com/pinecone-io/go-pinecone/v5/pinecone"
)

type Pinecone = pn.DBase
type LocalDbase = localCombinedDB.DbaseVector
type Database struct {
	Secelted int

	P     Pinecone
	Local LocalDbase
}

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
		return &Database{
			Secelted: 1,
			P: Pinecone{
				Client: pc,
				Cfg:    cfg,
			},
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
