package dataset

import (
	"math/rand"
	"sync/atomic"
	"time"
)

// Dataset represents a test dataset
type Dataset interface {
	// Name returns the dataset name
	Name() string

	// Dimension returns the vector dimension
	Dimension() int

	// Size returns the total number of vectors
	Size() int

	// GenerateVectors generates n vectors
	GenerateVectors(n int) [][]float32

	// GenerateQueryVectors generates n query vectors
	GenerateQueryVectors(n int) [][]float32
}

// RandomDataset generates random vectors
type RandomDataset struct {
	name      string
	dimension int
	size      int
	seedBase  int64
	seedInc   atomic.Int64
}

// NewRandomDataset creates a new random dataset
func NewRandomDataset(name string, dimension, size int, seed int64) *RandomDataset {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &RandomDataset{
		name:      name,
		dimension: dimension,
		size:      size,
		seedBase:  seed,
	}
}

// Name returns the dataset name
func (d *RandomDataset) Name() string {
	return d.name
}

// Dimension returns the vector dimension
func (d *RandomDataset) Dimension() int {
	return d.dimension
}

// Size returns the total number of vectors
func (d *RandomDataset) Size() int {
	return d.size
}

// GenerateVectors generates n random vectors (thread-safe)
func (d *RandomDataset) GenerateVectors(n int) [][]float32 {
	// Create a local rand instance with unique seed for thread safety
	localSeed := d.seedBase + d.seedInc.Add(1)
	rng := rand.New(rand.NewSource(localSeed))

	vectors := make([][]float32, n)
	for i := 0; i < n; i++ {
		vectors[i] = generateVectorWithRng(rng, d.dimension)
	}
	return vectors
}

// GenerateQueryVectors generates n random query vectors
func (d *RandomDataset) GenerateQueryVectors(n int) [][]float32 {
	return d.GenerateVectors(n)
}

func generateVectorWithRng(rng *rand.Rand, dimension int) []float32 {
	vec := make([]float32, dimension)
	for i := range vec {
		vec[i] = rng.Float32()
	}
	return normalizeVector(vec)
}

// normalizeVector normalizes a vector to unit length
func normalizeVector(vec []float32) []float32 {
	var sum float32
	for _, v := range vec {
		sum += v * v
	}
	if sum == 0 {
		return vec
	}
	norm := float32(1.0 / float64(sum))
	for i := range vec {
		vec[i] *= norm
	}
	return vec
}

// PresetDatasets contains predefined dataset configurations
var PresetDatasets = map[string]struct {
	Dimension int
	Size      int
}{
	"small":       {Dimension: 128, Size: 10000},
	"medium":      {Dimension: 128, Size: 100000},
	"large":       {Dimension: 128, Size: 1000000},
	"cohere-100k": {Dimension: 768, Size: 100000},
	"cohere-1m":   {Dimension: 768, Size: 1000000},
	"openai-50k":  {Dimension: 1536, Size: 50000},
	"openai-500k": {Dimension: 1536, Size: 500000},
}

// GetPresetDataset returns a preset dataset by name
func GetPresetDataset(name string, seed int64) Dataset {
	if preset, ok := PresetDatasets[name]; ok {
		return NewRandomDataset(name, preset.Dimension, preset.Size, seed)
	}
	// Default to small dataset
	return NewRandomDataset("small", 128, 10000, seed)
}
