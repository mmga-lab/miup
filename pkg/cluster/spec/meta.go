package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ClusterStatus represents the cluster status
type ClusterStatus string

const (
	StatusDeploying ClusterStatus = "deploying"
	StatusRunning   ClusterStatus = "running"
	StatusStopped   ClusterStatus = "stopped"
	StatusUpgrading ClusterStatus = "upgrading"
	StatusScaling   ClusterStatus = "scaling"
	StatusUnknown   ClusterStatus = "unknown"
)

// ClusterMeta contains cluster metadata
type ClusterMeta struct {
	Name          string        `json:"name"`
	Version       string        `json:"version"`
	Mode          DeployMode    `json:"mode"`
	Backend       BackendType   `json:"backend"`
	Status        ClusterStatus `json:"status"`
	MilvusVersion string        `json:"milvus_version"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`

	// Ports
	MilvusPort   int `json:"milvus_port"`
	EtcdPort     int `json:"etcd_port"`
	MinioPort    int `json:"minio_port"`
	MinioConsole int `json:"minio_console"`

	// Monitoring
	PrometheusPort int `json:"prometheus_port,omitempty"`
	GrafanaPort    int `json:"grafana_port,omitempty"`

	// Kubernetes specific options (only set when Backend is kubernetes)
	Kubeconfig  string `json:"kubeconfig,omitempty"`
	KubeContext string `json:"kube_context,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
}

// SaveMeta saves cluster metadata to a file
func SaveMeta(meta *ClusterMeta, path string) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// LoadMeta loads cluster metadata from a file
func LoadMeta(path string) (*ClusterMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var meta ClusterMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &meta, nil
}

// NewClusterMeta creates a new cluster metadata from specification
func NewClusterMeta(name string, spec *Specification, milvusVersion string) *ClusterMeta {
	meta := &ClusterMeta{
		Name:          name,
		Version:       "1.0",
		Mode:          spec.GetMode(),
		Backend:       BackendKubernetes,
		Status:        StatusDeploying,
		MilvusVersion: milvusVersion,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Set ports from specification
	if len(spec.MilvusServers) > 0 {
		meta.MilvusPort = spec.MilvusServers[0].Port
	}
	if len(spec.EtcdServers) > 0 {
		meta.EtcdPort = spec.EtcdServers[0].ClientPort
	}
	if len(spec.MinioServers) > 0 {
		meta.MinioPort = spec.MinioServers[0].Port
		meta.MinioConsole = spec.MinioServers[0].ConsolePort
	}
	if len(spec.MonitorServers) > 0 {
		meta.PrometheusPort = spec.MonitorServers[0].PrometheusPort
	}
	if len(spec.GrafanaServers) > 0 {
		meta.GrafanaPort = spec.GrafanaServers[0].Port
	}

	return meta
}
