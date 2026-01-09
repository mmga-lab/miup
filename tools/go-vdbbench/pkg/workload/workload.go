package workload

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zilliztech/go-vdbbench/pkg/database"
	"github.com/zilliztech/go-vdbbench/pkg/dataset"
	"github.com/zilliztech/go-vdbbench/pkg/metrics"
)

// Config holds workload configuration
type Config struct {
	// Common settings
	Threads     int
	Duration    time.Duration
	Collection  string

	// Data settings
	Dataset     dataset.Dataset
	BatchSize   int

	// Search settings
	TopK        int

	// Index settings
	IndexType   string
	IndexParams map[string]interface{}
}

// DefaultConfig returns default workload configuration
func DefaultConfig() *Config {
	return &Config{
		Threads:    10,
		Duration:   60 * time.Second,
		Collection: "benchmark_collection",
		BatchSize:  1000,
		TopK:       10,
		IndexType:  "IVF_FLAT",
		IndexParams: map[string]interface{}{
			"nlist": 1024,
		},
	}
}

// Workload represents a benchmark workload
type Workload struct {
	db        database.VectorDB
	config    *Config
	collector *metrics.Collector
}

// NewWorkload creates a new workload
func NewWorkload(db database.VectorDB, config *Config) *Workload {
	return &Workload{
		db:        db,
		config:    config,
		collector: metrics.NewCollector(),
	}
}

// Prepare prepares the collection and data for benchmark
func (w *Workload) Prepare(ctx context.Context, progressFn func(current, total int)) error {
	cfg := w.config
	ds := cfg.Dataset

	// Check if collection exists
	exists, err := w.db.HasCollection(ctx, cfg.Collection)
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}

	// Drop if exists
	if exists {
		if err := w.db.DropCollection(ctx, cfg.Collection); err != nil {
			return fmt.Errorf("failed to drop collection: %w", err)
		}
	}

	// Create collection
	if err := w.db.CreateCollection(ctx, cfg.Collection, ds.Dimension(), "L2"); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// Insert data in batches
	totalVectors := ds.Size()
	inserted := 0

	for inserted < totalVectors {
		batchSize := cfg.BatchSize
		if inserted+batchSize > totalVectors {
			batchSize = totalVectors - inserted
		}

		vectors := ds.GenerateVectors(batchSize)
		if err := w.db.Insert(ctx, cfg.Collection, vectors); err != nil {
			return fmt.Errorf("failed to insert batch: %w", err)
		}

		inserted += batchSize
		if progressFn != nil {
			progressFn(inserted, totalVectors)
		}
	}

	// Create index
	if err := w.db.CreateIndex(ctx, cfg.Collection, cfg.IndexType, cfg.IndexParams); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Load collection
	if err := w.db.LoadCollection(ctx, cfg.Collection); err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	return nil
}

// RunSearch runs search workload
func (w *Workload) RunSearch(ctx context.Context, progressFn func(ops int64, elapsed time.Duration)) *metrics.Result {
	cfg := w.config
	ds := cfg.Dataset

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, cfg.Duration)
	defer cancel()

	var wg sync.WaitGroup
	var totalOps int64

	w.collector.Start()

	// Start workers
	for i := 0; i < cfg.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Generate query vector
					queryVectors := ds.GenerateQueryVectors(1)

					// Execute search
					start := time.Now()
					_, err := w.db.Search(ctx, cfg.Collection, queryVectors, cfg.TopK)
					latency := time.Since(start)

					if err != nil {
						w.collector.RecordError()
					} else {
						w.collector.Record(latency)
					}

					atomic.AddInt64(&totalOps, 1)
				}
			}
		}()
	}

	// Progress reporting
	if progressFn != nil {
		ticker := time.NewTicker(time.Second)
		go func() {
			for {
				select {
				case <-ctx.Done():
					ticker.Stop()
					return
				case <-ticker.C:
					ops, _, elapsed := w.collector.CurrentStats()
					progressFn(ops, elapsed)
				}
			}
		}()
	}

	wg.Wait()
	w.collector.Stop()

	return w.collector.Calculate()
}

// RunInsert runs insert workload
func (w *Workload) RunInsert(ctx context.Context, progressFn func(ops int64, elapsed time.Duration)) *metrics.Result {
	cfg := w.config
	ds := cfg.Dataset

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, cfg.Duration)
	defer cancel()

	var wg sync.WaitGroup

	w.collector.Start()

	// Start workers
	for i := 0; i < cfg.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Generate vectors
					vectors := ds.GenerateVectors(cfg.BatchSize)

					// Execute insert
					start := time.Now()
					err := w.db.Insert(ctx, cfg.Collection, vectors)
					latency := time.Since(start)

					if err != nil {
						w.collector.RecordError()
					} else {
						w.collector.Record(latency)
					}
				}
			}
		}()
	}

	// Progress reporting
	if progressFn != nil {
		ticker := time.NewTicker(time.Second)
		go func() {
			for {
				select {
				case <-ctx.Done():
					ticker.Stop()
					return
				case <-ticker.C:
					ops, _, elapsed := w.collector.CurrentStats()
					progressFn(ops, elapsed)
				}
			}
		}()
	}

	wg.Wait()
	w.collector.Stop()

	return w.collector.Calculate()
}

// Cleanup cleans up the benchmark collection
func (w *Workload) Cleanup(ctx context.Context) error {
	return w.db.DropCollection(ctx, w.config.Collection)
}
