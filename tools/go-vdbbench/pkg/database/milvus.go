package database

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// MilvusDB implements VectorDB interface for Milvus
type MilvusDB struct {
	config Config
	client client.Client
}

// NewMilvusDB creates a new Milvus database adapter
func NewMilvusDB(config Config) *MilvusDB {
	return &MilvusDB{
		config: config,
	}
}

// Name returns the database name
func (m *MilvusDB) Name() string {
	return "milvus"
}

// Connect connects to Milvus
func (m *MilvusDB) Connect(ctx context.Context) error {
	cfg := client.Config{
		Address: m.config.URI,
	}

	if m.config.Username != "" {
		cfg.Username = m.config.Username
		cfg.Password = m.config.Password
	}

	if m.config.Database != "" {
		cfg.DBName = m.config.Database
	}

	c, err := client.NewClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to Milvus: %w", err)
	}

	m.client = c
	return nil
}

// Close closes the connection
func (m *MilvusDB) Close() error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

// CreateCollection creates a collection
func (m *MilvusDB) CreateCollection(ctx context.Context, name string, dim int, metricType string) error {
	// Define schema
	schema := &entity.Schema{
		CollectionName: name,
		AutoID:         true,
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeInt64,
				PrimaryKey: true,
				AutoID:     true,
			},
			{
				Name:     "vector",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": fmt.Sprintf("%d", dim),
				},
			},
		},
	}

	// Create collection
	err := m.client.CreateCollection(ctx, schema, entity.DefaultShardNumber)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

// DropCollection drops a collection
func (m *MilvusDB) DropCollection(ctx context.Context, name string) error {
	return m.client.DropCollection(ctx, name)
}

// HasCollection checks if a collection exists
func (m *MilvusDB) HasCollection(ctx context.Context, name string) (bool, error) {
	return m.client.HasCollection(ctx, name)
}

// Insert inserts vectors into a collection
func (m *MilvusDB) Insert(ctx context.Context, collection string, vectors [][]float32) error {
	// Convert to entity.Column
	vectorColumn := entity.NewColumnFloatVector("vector", len(vectors[0]), vectors)

	_, err := m.client.Insert(ctx, collection, "", vectorColumn)
	if err != nil {
		return fmt.Errorf("failed to insert: %w", err)
	}

	return nil
}

// Search performs vector similarity search
func (m *MilvusDB) Search(ctx context.Context, collection string, vectors [][]float32, topK int) ([][]int64, error) {
	// Prepare search vectors
	searchVectors := make([]entity.Vector, len(vectors))
	for i, v := range vectors {
		searchVectors[i] = entity.FloatVector(v)
	}

	// Search parameters
	sp, _ := entity.NewIndexIvfFlatSearchParam(64) // nprobe=64

	results, err := m.client.Search(
		ctx,
		collection,
		nil,      // partitions
		"",       // expr
		[]string{"id"}, // output fields
		searchVectors,
		"vector",
		entity.L2,
		topK,
		sp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Extract IDs
	ids := make([][]int64, len(results))
	for i, result := range results {
		ids[i] = make([]int64, result.ResultCount)
		for j := 0; j < result.ResultCount; j++ {
			if idCol, ok := result.IDs.(*entity.ColumnInt64); ok {
				ids[i][j] = idCol.Data()[j]
			}
		}
	}

	return ids, nil
}

// CreateIndex creates an index on the collection
func (m *MilvusDB) CreateIndex(ctx context.Context, collection string, indexType string, params map[string]interface{}) error {
	var idx entity.Index
	var err error

	switch indexType {
	case "IVF_FLAT":
		nlist := 1024
		if v, ok := params["nlist"]; ok {
			nlist = v.(int)
		}
		idx, err = entity.NewIndexIvfFlat(entity.L2, nlist)
	case "HNSW":
		M := 16
		efConstruction := 256
		if v, ok := params["M"]; ok {
			M = v.(int)
		}
		if v, ok := params["efConstruction"]; ok {
			efConstruction = v.(int)
		}
		idx, err = entity.NewIndexHNSW(entity.L2, M, efConstruction)
	case "FLAT":
		idx, err = entity.NewIndexFlat(entity.L2)
	default:
		idx, err = entity.NewIndexIvfFlat(entity.L2, 1024)
	}

	if err != nil {
		return fmt.Errorf("failed to create index params: %w", err)
	}

	err = m.client.CreateIndex(ctx, collection, "vector", idx, false)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

// LoadCollection loads collection into memory
func (m *MilvusDB) LoadCollection(ctx context.Context, collection string) error {
	return m.client.LoadCollection(ctx, collection, false)
}

// GetCollectionStats returns collection statistics
func (m *MilvusDB) GetCollectionStats(ctx context.Context, collection string) (*CollectionStats, error) {
	stats, err := m.client.GetCollectionStatistics(ctx, collection)
	if err != nil {
		return nil, err
	}

	rowCount := int64(0)
	if v, ok := stats["row_count"]; ok {
		fmt.Sscanf(v, "%d", &rowCount)
	}

	return &CollectionStats{
		RowCount: rowCount,
	}, nil
}
