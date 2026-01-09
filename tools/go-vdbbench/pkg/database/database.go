package database

import (
	"context"
)

// VectorDB is the interface for vector database operations
type VectorDB interface {
	// Connect connects to the database
	Connect(ctx context.Context) error

	// Close closes the connection
	Close() error

	// CreateCollection creates a collection with the given schema
	CreateCollection(ctx context.Context, name string, dim int, metricType string) error

	// DropCollection drops a collection
	DropCollection(ctx context.Context, name string) error

	// HasCollection checks if a collection exists
	HasCollection(ctx context.Context, name string) (bool, error)

	// Insert inserts vectors into a collection
	Insert(ctx context.Context, collection string, vectors [][]float32) error

	// Search performs vector similarity search
	Search(ctx context.Context, collection string, vectors [][]float32, topK int) ([][]int64, error)

	// CreateIndex creates an index on the collection
	CreateIndex(ctx context.Context, collection string, indexType string, params map[string]interface{}) error

	// LoadCollection loads collection into memory
	LoadCollection(ctx context.Context, collection string) error

	// GetCollectionStats returns collection statistics
	GetCollectionStats(ctx context.Context, collection string) (*CollectionStats, error)

	// Name returns the database name
	Name() string
}

// CollectionStats holds collection statistics
type CollectionStats struct {
	RowCount int64
}

// Config holds database connection configuration
type Config struct {
	URI      string
	Username string
	Password string
	Database string
}
