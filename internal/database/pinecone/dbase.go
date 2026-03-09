package pinecone

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nikaydo/personal-assistant/internal/config"
	localCombinedDB "github.com/nikaydo/personal-assistant/internal/database/localCombinedDB"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/models"
	"github.com/pinecone-io/go-pinecone/v5/pinecone"
)

type DBase struct {
	Client    *pinecone.Client
	Index     *pinecone.Index
	IndexConn *pinecone.IndexConnection
	Cfg       *config.Config
}

func (db *DBase) CreateIndex() error {
	index, err := db.Client.CreateIndexForModel(context.Background(), &pinecone.CreateIndexForModelRequest{
		Name:   db.Cfg.PinecoreIndexName,
		Cloud:  pinecone.Cloud(db.Cfg.PinecoreCloud),
		Region: db.Cfg.PinecoreRegion,
		Embed: pinecone.CreateIndexForModelEmbed{
			Model: db.Cfg.PinecoreEmbedModel,
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

func (db *DBase) EnsureReady() error {
	ctx := context.Background()
	for range 12 {
		desc, err := db.Client.DescribeIndex(ctx, db.Cfg.PinecoreIndexName)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				if err := db.CreateIndex(); err != nil {
					return err
				}
				time.Sleep(2 * time.Second)
				continue
			}
			return err
		}
		if desc.Status.Ready {
			db.Index = desc
			return db.IndexConnection()
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("index %s is not ready after 12 attempts", db.Cfg.PinecoreIndexName)
}

func (db *DBase) SaveSummary(id string, vector []float32, data models.SummarizeResponse) error {
	if db.IndexConn == nil {
		if err := db.EnsureReady(); err != nil {
			return err
		}
	}
	if id == "" {
		return fmt.Errorf("id is required")
	}
	if len(vector) == 0 {
		return fmt.Errorf("vector is required")
	}

	metaMap := map[string]any{
		"id":         id,
		"category":   data.Category,
		"goal":       data.Goal,
		"importance": data.Importance,
		"status":     data.Status,
		"text":       data.Text,
		"created_at": time.Now().Unix(),
		"updated_at": time.Now().Unix(),
	}
	meta, err := pinecone.NewMetadata(metaMap)
	if err != nil {
		return fmt.Errorf("create pinecone metadata: %w", err)
	}

	values := make([]float32, len(vector))
	copy(values, vector)
	_, err = db.IndexConn.UpsertVectors(context.Background(), []*pinecone.Vector{{
		Id:       id,
		Values:   &values,
		Metadata: meta,
	}})
	if err != nil {
		return fmt.Errorf("upsert pinecone vector: %w", err)
	}
	return nil
}

type SearchMatch struct {
	ID        string
	Score     float32
	Data      models.SummarizeResponse
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (db *DBase) GetSummary(id string) (models.SummarizeResponse, time.Time, time.Time, bool, error) {
	if db.IndexConn == nil {
		if err := db.EnsureReady(); err != nil {
			return models.SummarizeResponse{}, time.Time{}, time.Time{}, false, err
		}
	}
	resp, err := db.IndexConn.FetchVectors(context.Background(), []string{id})
	if err != nil {
		return models.SummarizeResponse{}, time.Time{}, time.Time{}, false, fmt.Errorf("fetch pinecone vector: %w", err)
	}
	vec, ok := resp.Vectors[id]
	if !ok || vec == nil {
		return models.SummarizeResponse{}, time.Time{}, time.Time{}, false, nil
	}
	data, createdAt, updatedAt := summaryFromMetadata(vec.Metadata)
	return data, createdAt, updatedAt, true, nil
}

func (db *DBase) SearchByVector(vector []float32, topK int, filters localCombinedDB.Filters) ([]SearchMatch, error) {
	if db.IndexConn == nil {
		if err := db.EnsureReady(); err != nil {
			return nil, err
		}
	}
	if len(vector) == 0 {
		return nil, fmt.Errorf("vector is required")
	}
	if topK <= 0 {
		topK = 10
	}

	req := &pinecone.QueryByVectorValuesRequest{
		Vector:          vector,
		TopK:            uint32(topK),
		IncludeMetadata: true,
	}
	if filter := buildPineconeFilter(filters); len(filter) > 0 {
		mf, err := pinecone.NewMetadataFilter(filter)
		if err != nil {
			return nil, fmt.Errorf("build metadata filter: %w", err)
		}
		req.MetadataFilter = mf
	}

	resp, err := db.IndexConn.QueryByVectorValues(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("query pinecone by vector: %w", err)
	}

	out := make([]SearchMatch, 0, len(resp.Matches))
	for _, match := range resp.Matches {
		if match == nil || match.Vector == nil {
			continue
		}
		data, createdAt, updatedAt := summaryFromMetadata(match.Vector.Metadata)
		out = append(out, SearchMatch{
			ID:        match.Vector.Id,
			Score:     match.Score,
			Data:      data,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
	return out, nil
}

func (db *DBase) SearchByFilters(filters localCombinedDB.Filters) ([]SearchMatch, error) {
	if db.IndexConn == nil {
		if err := db.EnsureReady(); err != nil {
			return nil, err
		}
	}
	filterMap := buildPineconeFilter(filters)
	if len(filterMap) == 0 {
		return nil, fmt.Errorf("at least one filter is required for pinecone metadata search")
	}
	mf, err := pinecone.NewMetadataFilter(filterMap)
	if err != nil {
		return nil, fmt.Errorf("build metadata filter: %w", err)
	}

	limit := uint32(filters.Limit)
	if limit == 0 {
		limit = 50
	}
	resp, err := db.IndexConn.FetchVectorsByMetadata(context.Background(), &pinecone.FetchVectorsByMetadataRequest{
		Filter: mf,
		Limit:  &limit,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch pinecone by metadata: %w", err)
	}

	out := make([]SearchMatch, 0, len(resp.Vectors))
	for id, vec := range resp.Vectors {
		if vec == nil {
			continue
		}
		data, createdAt, updatedAt := summaryFromMetadata(vec.Metadata)
		out = append(out, SearchMatch{
			ID:        id,
			Score:     0,
			Data:      data,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
	return out, nil
}

func buildPineconeFilter(filters localCombinedDB.Filters) map[string]any {
	items := make([]map[string]any, 0, 6)
	if len(filters.IDs) > 0 {
		items = append(items, map[string]any{"id": map[string]any{"$in": filters.IDs}})
	}
	if filters.Category != "" {
		items = append(items, map[string]any{"category": map[string]any{"$eq": filters.Category}})
	}
	if filters.Goal != "" {
		items = append(items, map[string]any{"goal": map[string]any{"$eq": filters.Goal}})
	}
	if filters.Importance != "" {
		items = append(items, map[string]any{"importance": map[string]any{"$eq": filters.Importance}})
	}
	if filters.Status != "" {
		items = append(items, map[string]any{"status": map[string]any{"$eq": filters.Status}})
	}
	if filters.TextQuery != "" {
		items = append(items, map[string]any{"text": map[string]any{"$eq": filters.TextQuery}})
	}

	if len(items) == 0 {
		return map[string]any{}
	}
	if len(items) == 1 {
		return items[0]
	}
	return map[string]any{"$and": items}
}

func summaryFromMetadata(meta *pinecone.Metadata) (models.SummarizeResponse, time.Time, time.Time) {
	out := models.SummarizeResponse{}
	if meta == nil {
		return out, time.Time{}, time.Time{}
	}
	m := meta.AsMap()
	out.Category = toString(m["category"])
	out.Goal = toString(m["goal"])
	out.Importance = toString(m["importance"])
	out.Status = toString(m["status"])
	out.Text = toString(m["text"])
	createdAt := fromUnixAny(m["created_at"])
	updatedAt := fromUnixAny(m["updated_at"])
	return out, createdAt, updatedAt
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return strings.Trim(string(b), "\"")
	}
}

func fromUnixAny(v any) time.Time {
	switch t := v.(type) {
	case float64:
		return time.Unix(int64(t), 0)
	case int64:
		return time.Unix(t, 0)
	case int:
		return time.Unix(int64(t), 0)
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return time.Time{}
		}
		return time.Unix(n, 0)
	default:
		return time.Time{}
	}
}
