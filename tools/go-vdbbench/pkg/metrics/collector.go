package metrics

import (
	"sort"
	"sync"
	"time"
)

// Collector collects and calculates benchmark metrics
type Collector struct {
	mu        sync.Mutex
	latencies []time.Duration
	errors    int64
	startTime time.Time
	endTime   time.Time
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		latencies: make([]time.Duration, 0, 10000),
	}
}

// Start marks the start of benchmark
func (c *Collector) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.startTime = time.Now()
}

// Stop marks the end of benchmark
func (c *Collector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.endTime = time.Now()
}

// Record records a single operation latency
func (c *Collector) Record(latency time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.latencies = append(c.latencies, latency)
}

// RecordError records an error
func (c *Collector) RecordError() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors++
}

// Result represents benchmark results
type Result struct {
	TotalOps   int64
	Duration   time.Duration
	QPS        float64
	AvgLatency time.Duration
	MinLatency time.Duration
	MaxLatency time.Duration
	P50Latency time.Duration
	P95Latency time.Duration
	P99Latency time.Duration
	Errors     int64
	ErrorRate  float64
}

// Calculate calculates the final metrics
func (c *Collector) Calculate() *Result {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.latencies) == 0 {
		return &Result{}
	}

	// Sort latencies for percentile calculation
	sorted := make([]time.Duration, len(c.latencies))
	copy(sorted, c.latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate total and average
	var total time.Duration
	for _, l := range sorted {
		total += l
	}

	n := len(sorted)
	duration := c.endTime.Sub(c.startTime)
	totalOps := int64(n)

	result := &Result{
		TotalOps:   totalOps,
		Duration:   duration,
		QPS:        float64(totalOps) / duration.Seconds(),
		AvgLatency: total / time.Duration(n),
		MinLatency: sorted[0],
		MaxLatency: sorted[n-1],
		P50Latency: sorted[n*50/100],
		P95Latency: sorted[n*95/100],
		P99Latency: sorted[n*99/100],
		Errors:     c.errors,
		ErrorRate:  float64(c.errors) / float64(totalOps+c.errors) * 100,
	}

	return result
}

// CurrentStats returns current statistics (for progress display)
func (c *Collector) CurrentStats() (ops int64, errors int64, elapsed time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return int64(len(c.latencies)), c.errors, time.Since(c.startTime)
}
