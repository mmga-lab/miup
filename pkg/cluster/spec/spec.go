package spec

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DeployMode represents the deployment mode
type DeployMode string

const (
	ModeStandalone   DeployMode = "standalone"
	ModeDistributed  DeployMode = "distributed"
	ModeCluster      DeployMode = "cluster" // Alias for distributed (backward compatibility)
)

// BackendType represents the deployment backend
type BackendType string

const (
	BackendLocal      BackendType = "local"
	BackendKubernetes BackendType = "kubernetes"
)

// Specification represents a cluster topology specification
type Specification struct {
	Global         GlobalOptions    `yaml:"global"`
	ServerConfigs  ServerConfigs    `yaml:"server_configs,omitempty"`
	MilvusServers  []MilvusSpec     `yaml:"milvus_servers"`
	EtcdServers    []EtcdSpec       `yaml:"etcd_servers"`
	MinioServers   []MinioSpec      `yaml:"minio_servers"`
	PulsarServers  []PulsarSpec     `yaml:"pulsar_servers,omitempty"`
	MonitorServers []MonitorSpec    `yaml:"monitoring_servers,omitempty"`
	GrafanaServers []GrafanaSpec    `yaml:"grafana_servers,omitempty"`
}

// GlobalOptions contains global configuration
type GlobalOptions struct {
	User      string `yaml:"user,omitempty"`
	SSHPort   int    `yaml:"ssh_port,omitempty"`
	DeployDir string `yaml:"deploy_dir,omitempty"`
	DataDir   string `yaml:"data_dir,omitempty"`
	LogDir    string `yaml:"log_dir,omitempty"`

	// TLS configuration
	TLS TLSConfig `yaml:"tls,omitempty"`

	// Kubernetes specific
	Namespace    string `yaml:"namespace,omitempty"`
	StorageClass string `yaml:"storage_class,omitempty"`
}

// TLSConfig contains TLS configuration
type TLSConfig struct {
	// Enabled enables TLS for client connections
	Enabled bool `yaml:"enabled,omitempty"`

	// Mode specifies TLS mode: 1 for one-way (server cert only), 2 for two-way (mutual TLS)
	Mode int `yaml:"mode,omitempty"`

	// CertFile is the path to the server certificate file (server.pem)
	CertFile string `yaml:"cert_file,omitempty"`

	// KeyFile is the path to the server private key file (server.key)
	KeyFile string `yaml:"key_file,omitempty"`

	// CAFile is the path to the CA certificate file (ca.pem)
	CAFile string `yaml:"ca_file,omitempty"`

	// InternalEnabled enables TLS for internal component communication
	InternalEnabled bool `yaml:"internal_enabled,omitempty"`

	// SecretName is the Kubernetes secret name containing TLS certificates (for K8s deployment)
	SecretName string `yaml:"secret_name,omitempty"`
}

// ServerConfigs contains server configuration overrides
type ServerConfigs struct {
	Milvus map[string]any `yaml:"milvus,omitempty"`
	Etcd   map[string]any `yaml:"etcd,omitempty"`
	Minio  map[string]any `yaml:"minio,omitempty"`
}

// MilvusSpec represents Milvus server specification
type MilvusSpec struct {
	Host       string           `yaml:"host"`
	Port       int              `yaml:"port,omitempty"`
	Mode       DeployMode       `yaml:"mode,omitempty"`
	Components MilvusComponents `yaml:"components,omitempty"`
	Config     map[string]any   `yaml:"config,omitempty"`
}

// MilvusComponents represents Milvus component configuration
type MilvusComponents struct {
	RootCoord  ComponentSpec `yaml:"rootCoord,omitempty"`
	QueryCoord ComponentSpec `yaml:"queryCoord,omitempty"`
	DataCoord  ComponentSpec `yaml:"dataCoord,omitempty"`
	IndexCoord ComponentSpec `yaml:"indexCoord,omitempty"`
	Proxy      ComponentSpec `yaml:"proxy,omitempty"`
	QueryNode  ComponentSpec `yaml:"queryNode,omitempty"`
	DataNode   ComponentSpec `yaml:"dataNode,omitempty"`
	IndexNode  ComponentSpec `yaml:"indexNode,omitempty"`
}

// ComponentSpec represents a component specification
type ComponentSpec struct {
	Replicas  int          `yaml:"replicas,omitempty"`
	Resources ResourceSpec `yaml:"resources,omitempty"`
}

// ResourceSpec represents resource requirements
type ResourceSpec struct {
	CPU     string `yaml:"cpu,omitempty"`
	Memory  string `yaml:"memory,omitempty"`
	Storage string `yaml:"storage,omitempty"`
}

// EtcdSpec represents etcd server specification
type EtcdSpec struct {
	Host       string `yaml:"host"`
	ClientPort int    `yaml:"client_port,omitempty"`
	PeerPort   int    `yaml:"peer_port,omitempty"`
	DataDir    string `yaml:"data_dir,omitempty"`
}

// MinioSpec represents MinIO server specification
type MinioSpec struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port,omitempty"`
	ConsolePort int    `yaml:"console_port,omitempty"`
	AccessKey   string `yaml:"access_key,omitempty"`
	SecretKey   string `yaml:"secret_key,omitempty"`
	Bucket      string `yaml:"bucket,omitempty"`
	DataDir     string `yaml:"data_dir,omitempty"`
}

// PulsarSpec represents Pulsar server specification
type PulsarSpec struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port,omitempty"`
	HTTPPort int    `yaml:"http_port,omitempty"`
}

// MonitorSpec represents monitoring server specification
type MonitorSpec struct {
	Host             string `yaml:"host"`
	PrometheusPort   int    `yaml:"prometheus_port,omitempty"`
	AlertmanagerPort int    `yaml:"alertmanager_port,omitempty"`
	DataDir          string `yaml:"data_dir,omitempty"`
	Retention        string `yaml:"retention,omitempty"`
}

// GrafanaSpec represents Grafana server specification
type GrafanaSpec struct {
	Host          string `yaml:"host"`
	Port          int    `yaml:"port,omitempty"`
	AdminPassword string `yaml:"admin_password,omitempty"`
}

// LoadSpecification loads a specification from a YAML file
func LoadSpecification(path string) (*Specification, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read topology file: %w", err)
	}

	var spec Specification
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse topology file: %w", err)
	}

	// Set defaults
	spec.setDefaults()

	return &spec, nil
}

// setDefaults sets default values for the specification
func (s *Specification) setDefaults() {
	// Global defaults
	if s.Global.DeployDir == "" {
		s.Global.DeployDir = "/opt/milvus"
	}
	if s.Global.DataDir == "" {
		s.Global.DataDir = "/data/milvus"
	}
	if s.Global.LogDir == "" {
		s.Global.LogDir = "/var/log/milvus"
	}
	if s.Global.Namespace == "" {
		s.Global.Namespace = "milvus"
	}

	// Milvus defaults
	for i := range s.MilvusServers {
		if s.MilvusServers[i].Port == 0 {
			s.MilvusServers[i].Port = 19530
		}
		if s.MilvusServers[i].Mode == "" {
			s.MilvusServers[i].Mode = ModeStandalone
		}
		setComponentDefaults(&s.MilvusServers[i].Components)
	}

	// Etcd defaults
	for i := range s.EtcdServers {
		if s.EtcdServers[i].ClientPort == 0 {
			s.EtcdServers[i].ClientPort = 2379
		}
		if s.EtcdServers[i].PeerPort == 0 {
			s.EtcdServers[i].PeerPort = 2380
		}
	}

	// MinIO defaults
	for i := range s.MinioServers {
		if s.MinioServers[i].Port == 0 {
			s.MinioServers[i].Port = 9000
		}
		if s.MinioServers[i].ConsolePort == 0 {
			s.MinioServers[i].ConsolePort = 9001
		}
		if s.MinioServers[i].AccessKey == "" {
			s.MinioServers[i].AccessKey = "minioadmin"
		}
		if s.MinioServers[i].SecretKey == "" {
			s.MinioServers[i].SecretKey = "minioadmin"
		}
		if s.MinioServers[i].Bucket == "" {
			s.MinioServers[i].Bucket = "milvus-bucket"
		}
	}

	// Monitor defaults
	for i := range s.MonitorServers {
		if s.MonitorServers[i].PrometheusPort == 0 {
			s.MonitorServers[i].PrometheusPort = 9090
		}
		if s.MonitorServers[i].AlertmanagerPort == 0 {
			s.MonitorServers[i].AlertmanagerPort = 9093
		}
	}

	// Grafana defaults
	for i := range s.GrafanaServers {
		if s.GrafanaServers[i].Port == 0 {
			s.GrafanaServers[i].Port = 3000
		}
		if s.GrafanaServers[i].AdminPassword == "" {
			s.GrafanaServers[i].AdminPassword = "admin"
		}
	}
}

func setComponentDefaults(c *MilvusComponents) {
	if c.Proxy.Replicas == 0 {
		c.Proxy.Replicas = 1
	}
	if c.RootCoord.Replicas == 0 {
		c.RootCoord.Replicas = 1
	}
	if c.QueryCoord.Replicas == 0 {
		c.QueryCoord.Replicas = 1
	}
	if c.DataCoord.Replicas == 0 {
		c.DataCoord.Replicas = 1
	}
	if c.IndexCoord.Replicas == 0 {
		c.IndexCoord.Replicas = 1
	}
	if c.QueryNode.Replicas == 0 {
		c.QueryNode.Replicas = 1
	}
	if c.DataNode.Replicas == 0 {
		c.DataNode.Replicas = 1
	}
	if c.IndexNode.Replicas == 0 {
		c.IndexNode.Replicas = 1
	}
}

// Validate validates the specification
func (s *Specification) Validate() error {
	if len(s.MilvusServers) == 0 {
		return fmt.Errorf("at least one milvus server is required")
	}
	if len(s.EtcdServers) == 0 {
		return fmt.Errorf("at least one etcd server is required")
	}
	if len(s.MinioServers) == 0 {
		return fmt.Errorf("at least one minio server is required")
	}

	// Validate hosts
	for i, server := range s.MilvusServers {
		if server.Host == "" {
			return fmt.Errorf("milvus_servers[%d].host is required", i)
		}
	}
	for i, server := range s.EtcdServers {
		if server.Host == "" {
			return fmt.Errorf("etcd_servers[%d].host is required", i)
		}
	}
	for i, server := range s.MinioServers {
		if server.Host == "" {
			return fmt.Errorf("minio_servers[%d].host is required", i)
		}
	}

	// Validate TLS configuration
	if s.Global.TLS.Enabled {
		// For local deployment, cert files are required
		// For K8s deployment, either cert files or secret name is required
		if s.Global.TLS.SecretName == "" {
			if s.Global.TLS.CertFile == "" {
				return fmt.Errorf("tls.cert_file is required when TLS is enabled")
			}
			if s.Global.TLS.KeyFile == "" {
				return fmt.Errorf("tls.key_file is required when TLS is enabled")
			}
		}
		// Validate TLS mode
		if s.Global.TLS.Mode != 0 && s.Global.TLS.Mode != 1 && s.Global.TLS.Mode != 2 {
			return fmt.Errorf("tls.mode must be 1 (one-way) or 2 (two-way)")
		}
	}

	return nil
}

// GetMode returns the deployment mode based on the specification
func (s *Specification) GetMode() DeployMode {
	if len(s.MilvusServers) == 0 {
		return ModeStandalone
	}
	return s.MilvusServers[0].Mode
}

// IsDistributed returns true if the mode is distributed (or cluster for backward compatibility)
func (s *Specification) IsDistributed() bool {
	mode := s.GetMode()
	return mode == ModeDistributed || mode == ModeCluster
}

// HasMonitoring returns true if monitoring is enabled
func (s *Specification) HasMonitoring() bool {
	return len(s.MonitorServers) > 0 || len(s.GrafanaServers) > 0
}

// HasTLS returns true if TLS is enabled
func (s *Specification) HasTLS() bool {
	return s.Global.TLS.Enabled
}

// GetTLSMode returns the TLS mode (1 for one-way, 2 for two-way)
func (s *Specification) GetTLSMode() int {
	if s.Global.TLS.Mode == 0 {
		return 1 // Default to one-way TLS
	}
	return s.Global.TLS.Mode
}

// SaveSpecification saves a specification to a YAML file
func SaveSpecification(spec *Specification, path string) error {
	data, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal specification: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write specification: %w", err)
	}

	return nil
}
