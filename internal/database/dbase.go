package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/pinecone-io/go-pinecone/v5/pinecone"
)

type DBase struct {
	Client    *pinecone.Client
	Index     *pinecone.Index
	IndexConn *pinecone.IndexConnection
	Cfg       *config.Config
}

func InitDB(apikey string, cfg *config.Config) (*DBase, error) {
	pc, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: apikey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &DBase{
		Client: pc,
		Cfg:    cfg,
	}, nil
}

func (db *DBase) CreateIndex() error {
	index, err := db.Client.CreateIndexForModel(context.Background(), &pinecone.CreateIndexForModelRequest{
		Name:   db.Cfg.IndexName,
		Cloud:  pinecone.Cloud(db.Cfg.Cloud),
		Region: db.Cfg.Region,
		Embed: pinecone.CreateIndexForModelEmbed{
			Model: db.Cfg.EmbedModel,
			FieldMap: map[string]any{
				"text":       "text",
				"category":   "category",
				"goal":       "goal",
				"importance": "float",
				"timestamp":  "timestamp",
				"status":     "status",
			},
		},
	})
	if err != nil {
		return err
	}
	db.Index = index
	return nil
}

func (db *DBase) WaitIndexReady(name string, log *logg.Logger) error {
	ctx := context.Background()
	for range 12 {
		desc, err := db.Client.DescribeIndex(ctx, name)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				if err := db.CreateIndex(); err != nil {
					return err
				}
				if err := db.WaitIndexReady(name, log); err != nil {
					return err
				}
				return nil
			}
			return err
		}

		if desc.Status.Ready {
			log.Info("Index is ready")
			db.Index = desc
			if err := db.IndexConnection(); err != nil {
				return err
			}
			return nil
		}
		log.Info("Index not ready...")
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("index %s is not ready after 12 attempts", name)
}

func (db *DBase) IndexConnection() error {
	idxConnection, err := db.Client.Index(pinecone.NewIndexConnParams{Host: db.Index.Host})
	if err != nil {
		return err
	}
	db.IndexConn = idxConnection
	return nil
}

func (db *DBase) Upsert(records []*pinecone.IntegratedRecord) error {
	return db.IndexConn.UpsertRecords(context.Background(), records)
}
