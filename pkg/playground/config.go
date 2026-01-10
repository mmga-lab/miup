package playground

// Mode represents the Milvus deployment mode
type Mode string

const (
	ModeStandalone Mode = "standalone"
)

// Config holds the playground configuration
type Config struct {
	// Tag is the unique identifier for this playground instance
	Tag string

	// Mode is the Milvus deployment mode (standalone only)
	Mode Mode

	// MilvusVersion is the Milvus version to use
	MilvusVersion string

	// EtcdVersion is the etcd version to use
	EtcdVersion string

	// MinioVersion is the MinIO version to use
	MinioVersion string

	// WithMonitor enables Prometheus and Grafana
	WithMonitor bool

	// Ports configuration
	MilvusPort     int
	EtcdPort       int
	MinioPort      int
	MinioConsole   int
	PrometheusPort int
	GrafanaPort    int
}

// DefaultConfig returns the default playground configuration
func DefaultConfig() *Config {
	return &Config{
		Tag:            "default",
		Mode:           ModeStandalone,
		MilvusVersion:  "v2.5.4",
		EtcdVersion:    "3.5.18",
		MinioVersion:   "RELEASE.2023-03-20T20-16-18Z",
		WithMonitor:    false,
		MilvusPort:     19530,
		EtcdPort:       2379,
		MinioPort:      9000,
		MinioConsole:   9001,
		PrometheusPort: 9090,
		GrafanaPort:    3000,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Tag == "" {
		c.Tag = "default"
	}
	if c.Mode == "" {
		c.Mode = ModeStandalone
	}
	if c.MilvusVersion == "" {
		c.MilvusVersion = "v2.5.4"
	}
	return nil
}
