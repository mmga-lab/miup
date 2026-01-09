package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/zilliztech/go-vdbbench/pkg/database"
	"github.com/zilliztech/go-vdbbench/pkg/dataset"
	"github.com/zilliztech/go-vdbbench/pkg/metrics"
	"github.com/zilliztech/go-vdbbench/pkg/workload"
)

var (
	version = "0.1.0"

	rootCmd = &cobra.Command{
		Use:   "go-vdbbench",
		Short: "Vector database benchmark tool",
		Long: `go-vdbbench is a vector database benchmark tool written in Go.

It supports benchmarking various vector databases including:
  - Milvus
  - (More databases coming soon)

Examples:
  go-vdbbench milvus search --uri localhost:19530 --dataset small
  go-vdbbench milvus insert --uri localhost:19530 --threads 10
  go-vdbbench milvus prepare --uri localhost:19530 --dataset cohere-100k`,
	}
)

func main() {
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newMilvusCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, color.RedString("Error: %v", err))
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("go-vdbbench version %s\n", version)
		},
	}
}

func newMilvusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "milvus",
		Short: "Benchmark Milvus vector database",
		Long: `Run benchmark tests against a Milvus instance.

Available commands:
  prepare   Prepare test data (create collection, insert data, build index)
  search    Run search performance test
  insert    Run insert performance test
  cleanup   Clean up test data`,
	}

	cmd.AddCommand(newMilvusPrepareCmd())
	cmd.AddCommand(newMilvusSearchCmd())
	cmd.AddCommand(newMilvusInsertCmd())
	cmd.AddCommand(newMilvusCleanupCmd())

	return cmd
}

// Common flags
type commonFlags struct {
	uri        string
	username   string
	password   string
	dbName     string
	collection string
	datasetName string
	dimension  int
	dataSize   int
	threads    int
	duration   int
	batchSize  int
	topK       int
	indexType  string
}

func addCommonFlags(cmd *cobra.Command, flags *commonFlags) {
	cmd.Flags().StringVar(&flags.uri, "uri", "localhost:19530", "Milvus server URI")
	cmd.Flags().StringVar(&flags.username, "username", "", "Username for authentication")
	cmd.Flags().StringVar(&flags.password, "password", "", "Password for authentication")
	cmd.Flags().StringVar(&flags.dbName, "db", "", "Database name")
	cmd.Flags().StringVar(&flags.collection, "collection", "benchmark_collection", "Collection name")
	cmd.Flags().StringVar(&flags.datasetName, "dataset", "small", "Dataset name (small, medium, large, cohere-100k, cohere-1m, openai-50k)")
	cmd.Flags().IntVar(&flags.dimension, "dimension", 0, "Vector dimension (overrides dataset default)")
	cmd.Flags().IntVar(&flags.dataSize, "size", 0, "Data size (overrides dataset default)")
	cmd.Flags().IntVar(&flags.threads, "threads", 10, "Number of concurrent threads")
	cmd.Flags().IntVar(&flags.duration, "duration", 60, "Test duration in seconds")
	cmd.Flags().IntVar(&flags.batchSize, "batch-size", 1000, "Batch size for insert")
	cmd.Flags().IntVar(&flags.topK, "top-k", 10, "Number of results for search")
	cmd.Flags().StringVar(&flags.indexType, "index-type", "IVF_FLAT", "Index type (FLAT, IVF_FLAT, HNSW)")
}

func createDBAndWorkload(flags *commonFlags) (database.VectorDB, *workload.Config) {
	// Create database
	db := database.NewMilvusDB(database.Config{
		URI:      flags.uri,
		Username: flags.username,
		Password: flags.password,
		Database: flags.dbName,
	})

	// Get dataset
	ds := dataset.GetPresetDataset(flags.datasetName, time.Now().UnixNano())

	// Override dataset settings if specified
	if flags.dimension > 0 || flags.dataSize > 0 {
		dim := ds.Dimension()
		size := ds.Size()
		if flags.dimension > 0 {
			dim = flags.dimension
		}
		if flags.dataSize > 0 {
			size = flags.dataSize
		}
		ds = dataset.NewRandomDataset(flags.datasetName, dim, size, time.Now().UnixNano())
	}

	// Create workload config
	cfg := workload.DefaultConfig()
	cfg.Threads = flags.threads
	cfg.Duration = time.Duration(flags.duration) * time.Second
	cfg.Collection = flags.collection
	cfg.Dataset = ds
	cfg.BatchSize = flags.batchSize
	cfg.TopK = flags.topK
	cfg.IndexType = flags.indexType

	return db, cfg
}

func newMilvusPrepareCmd() *cobra.Command {
	var flags commonFlags

	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare test data",
		Long: `Prepare test data for benchmarking.

This command will:
  1. Create a new collection
  2. Insert test vectors
  3. Build index
  4. Load collection into memory`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, cfg := createDBAndWorkload(&flags)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle interrupt
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			// Connect
			fmt.Printf("Connecting to Milvus at %s...\n", flags.uri)
			if err := db.Connect(ctx); err != nil {
				return err
			}
			defer db.Close()

			// Prepare
			w := workload.NewWorkload(db, cfg)

			fmt.Printf("Preparing dataset: %s (%d vectors, %d dimensions)\n",
				cfg.Dataset.Name(), cfg.Dataset.Size(), cfg.Dataset.Dimension())

			startTime := time.Now()
			err := w.Prepare(ctx, func(current, total int) {
				pct := float64(current) / float64(total) * 100
				fmt.Printf("\r  Inserting: %d/%d (%.1f%%)    ", current, total, pct)
			})
			if err != nil {
				return err
			}
			fmt.Println()

			elapsed := time.Since(startTime)
			fmt.Printf("\n%s Data prepared in %s\n", color.GreenString("✓"), elapsed.Round(time.Second))

			return nil
		},
	}

	addCommonFlags(cmd, &flags)
	return cmd
}

func newMilvusSearchCmd() *cobra.Command {
	var flags commonFlags

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Run search performance test",
		Long: `Run search performance test against Milvus.

The test will execute concurrent vector similarity searches and measure:
  - QPS (queries per second)
  - Latency (avg, p50, p95, p99)
  - Error rate`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, cfg := createDBAndWorkload(&flags)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle interrupt
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			// Connect
			fmt.Printf("Connecting to Milvus at %s...\n", flags.uri)
			if err := db.Connect(ctx); err != nil {
				return err
			}
			defer db.Close()

			// Print config
			printBenchConfig("Search", cfg)

			// Run benchmark
			w := workload.NewWorkload(db, cfg)
			result := w.RunSearch(ctx, func(ops int64, elapsed time.Duration) {
				qps := float64(ops) / elapsed.Seconds()
				fmt.Printf("\r  Running: %s | Ops: %d | QPS: %.1f    ", elapsed.Round(time.Second), ops, qps)
			})
			fmt.Println()

			// Print results
			printResults(result)
			return nil
		},
	}

	addCommonFlags(cmd, &flags)
	return cmd
}

func newMilvusInsertCmd() *cobra.Command {
	var flags commonFlags

	cmd := &cobra.Command{
		Use:   "insert",
		Short: "Run insert performance test",
		Long: `Run insert performance test against Milvus.

The test will execute concurrent batch inserts and measure:
  - Throughput (batches per second)
  - Latency (avg, p50, p95, p99)
  - Error rate`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, cfg := createDBAndWorkload(&flags)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle interrupt
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			// Connect
			fmt.Printf("Connecting to Milvus at %s...\n", flags.uri)
			if err := db.Connect(ctx); err != nil {
				return err
			}
			defer db.Close()

			// Print config
			printBenchConfig("Insert", cfg)

			// Run benchmark
			w := workload.NewWorkload(db, cfg)
			result := w.RunInsert(ctx, func(ops int64, elapsed time.Duration) {
				qps := float64(ops) / elapsed.Seconds()
				fmt.Printf("\r  Running: %s | Batches: %d | Batches/s: %.1f    ", elapsed.Round(time.Second), ops, qps)
			})
			fmt.Println()

			// Print results
			printResults(result)
			return nil
		},
	}

	addCommonFlags(cmd, &flags)
	return cmd
}

func newMilvusCleanupCmd() *cobra.Command {
	var flags commonFlags

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up test data",
		Long:  `Remove the benchmark collection and all test data.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, cfg := createDBAndWorkload(&flags)

			ctx := context.Background()

			// Connect
			fmt.Printf("Connecting to Milvus at %s...\n", flags.uri)
			if err := db.Connect(ctx); err != nil {
				return err
			}
			defer db.Close()

			// Cleanup
			w := workload.NewWorkload(db, cfg)
			if err := w.Cleanup(ctx); err != nil {
				return err
			}

			fmt.Printf("%s Collection '%s' dropped\n", color.GreenString("✓"), cfg.Collection)
			return nil
		},
	}

	addCommonFlags(cmd, &flags)
	return cmd
}

func printBenchConfig(testType string, cfg *workload.Config) {
	fmt.Println()
	fmt.Printf("%s Benchmark - %s\n", color.CyanString("Milvus"), testType)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Collection:  %s\n", cfg.Collection)
	fmt.Printf("Dataset:     %s (%d dim)\n", cfg.Dataset.Name(), cfg.Dataset.Dimension())
	fmt.Printf("Threads:     %d\n", cfg.Threads)
	fmt.Printf("Duration:    %s\n", cfg.Duration)
	if testType == "Search" {
		fmt.Printf("TopK:        %d\n", cfg.TopK)
	} else if testType == "Insert" {
		fmt.Printf("BatchSize:   %d\n", cfg.BatchSize)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

func printResults(result *metrics.Result) {
	fmt.Println()
	fmt.Println(color.GreenString("Results:"))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Total Ops:   %d\n", result.TotalOps)
	fmt.Printf("Duration:    %s\n", result.Duration.Round(time.Millisecond))
	fmt.Printf("QPS:         %.2f\n", result.QPS)
	fmt.Printf("Errors:      %d (%.2f%%)\n", result.Errors, result.ErrorRate)
	fmt.Println()
	fmt.Println("Latency:")
	fmt.Printf("  Min:       %s\n", result.MinLatency.Round(time.Microsecond))
	fmt.Printf("  Avg:       %s\n", result.AvgLatency.Round(time.Microsecond))
	fmt.Printf("  P50:       %s\n", result.P50Latency.Round(time.Microsecond))
	fmt.Printf("  P95:       %s\n", result.P95Latency.Round(time.Microsecond))
	fmt.Printf("  P99:       %s\n", result.P99Latency.Round(time.Microsecond))
	fmt.Printf("  Max:       %s\n", result.MaxLatency.Round(time.Microsecond))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
