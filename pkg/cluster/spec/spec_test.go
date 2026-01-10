package spec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate_EmptyMilvusServers(t *testing.T) {
	spec := &Specification{
		MilvusServers: []MilvusSpec{},
		EtcdServers:   []EtcdSpec{{Host: "localhost"}},
		MinioServers:  []MinioSpec{{Host: "localhost"}},
	}

	err := spec.Validate()
	if err == nil {
		t.Error("expected error for empty milvus servers")
	}
	if err.Error() != "at least one milvus server is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_EmptyEtcdServers(t *testing.T) {
	spec := &Specification{
		MilvusServers: []MilvusSpec{{Host: "localhost"}},
		EtcdServers:   []EtcdSpec{},
		MinioServers:  []MinioSpec{{Host: "localhost"}},
	}

	err := spec.Validate()
	if err == nil {
		t.Error("expected error for empty etcd servers")
	}
	if err.Error() != "at least one etcd server is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_EmptyMinioServers(t *testing.T) {
	spec := &Specification{
		MilvusServers: []MilvusSpec{{Host: "localhost"}},
		EtcdServers:   []EtcdSpec{{Host: "localhost"}},
		MinioServers:  []MinioSpec{},
	}

	err := spec.Validate()
	if err == nil {
		t.Error("expected error for empty minio servers")
	}
	if err.Error() != "at least one minio server is required" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidate_EmptyMilvusHost(t *testing.T) {
	spec := &Specification{
		MilvusServers: []MilvusSpec{{Host: ""}},
		EtcdServers:   []EtcdSpec{{Host: "localhost"}},
		MinioServers:  []MinioSpec{{Host: "localhost"}},
	}

	err := spec.Validate()
	if err == nil {
		t.Error("expected error for empty milvus host")
	}
}

func TestValidate_EmptyEtcdHost(t *testing.T) {
	spec := &Specification{
		MilvusServers: []MilvusSpec{{Host: "localhost"}},
		EtcdServers:   []EtcdSpec{{Host: ""}},
		MinioServers:  []MinioSpec{{Host: "localhost"}},
	}

	err := spec.Validate()
	if err == nil {
		t.Error("expected error for empty etcd host")
	}
}

func TestValidate_EmptyMinioHost(t *testing.T) {
	spec := &Specification{
		MilvusServers: []MilvusSpec{{Host: "localhost"}},
		EtcdServers:   []EtcdSpec{{Host: "localhost"}},
		MinioServers:  []MinioSpec{{Host: ""}},
	}

	err := spec.Validate()
	if err == nil {
		t.Error("expected error for empty minio host")
	}
}

func TestValidate_TLSEnabledWithoutCert(t *testing.T) {
	spec := &Specification{
		Global: GlobalOptions{
			TLS: TLSConfig{
				Enabled: true,
			},
		},
		MilvusServers: []MilvusSpec{{Host: "localhost"}},
		EtcdServers:   []EtcdSpec{{Host: "localhost"}},
		MinioServers:  []MinioSpec{{Host: "localhost"}},
	}

	err := spec.Validate()
	if err == nil {
		t.Error("expected error for TLS enabled without cert")
	}
}

func TestValidate_TLSEnabledWithoutKey(t *testing.T) {
	spec := &Specification{
		Global: GlobalOptions{
			TLS: TLSConfig{
				Enabled:  true,
				CertFile: "/path/to/cert.pem",
			},
		},
		MilvusServers: []MilvusSpec{{Host: "localhost"}},
		EtcdServers:   []EtcdSpec{{Host: "localhost"}},
		MinioServers:  []MinioSpec{{Host: "localhost"}},
	}

	err := spec.Validate()
	if err == nil {
		t.Error("expected error for TLS enabled without key")
	}
}

func TestValidate_TLSWithSecretName(t *testing.T) {
	spec := &Specification{
		Global: GlobalOptions{
			TLS: TLSConfig{
				Enabled:    true,
				SecretName: "milvus-tls-secret",
			},
		},
		MilvusServers: []MilvusSpec{{Host: "localhost"}},
		EtcdServers:   []EtcdSpec{{Host: "localhost"}},
		MinioServers:  []MinioSpec{{Host: "localhost"}},
	}

	err := spec.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_InvalidTLSMode(t *testing.T) {
	spec := &Specification{
		Global: GlobalOptions{
			TLS: TLSConfig{
				Enabled:    true,
				SecretName: "milvus-tls-secret",
				Mode:       3, // Invalid mode
			},
		},
		MilvusServers: []MilvusSpec{{Host: "localhost"}},
		EtcdServers:   []EtcdSpec{{Host: "localhost"}},
		MinioServers:  []MinioSpec{{Host: "localhost"}},
	}

	err := spec.Validate()
	if err == nil {
		t.Error("expected error for invalid TLS mode")
	}
}

func TestValidate_ValidSpec(t *testing.T) {
	spec := &Specification{
		MilvusServers: []MilvusSpec{{Host: "localhost", Port: 19530}},
		EtcdServers:   []EtcdSpec{{Host: "localhost"}},
		MinioServers:  []MinioSpec{{Host: "localhost"}},
	}

	err := spec.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSetDefaults(t *testing.T) {
	spec := &Specification{
		MilvusServers: []MilvusSpec{{Host: "localhost"}},
		EtcdServers:   []EtcdSpec{{Host: "localhost"}},
		MinioServers:  []MinioSpec{{Host: "localhost"}},
	}

	spec.setDefaults()

	// Check global defaults
	if spec.Global.DeployDir != "/opt/milvus" {
		t.Errorf("expected deploy_dir /opt/milvus, got %s", spec.Global.DeployDir)
	}
	if spec.Global.Namespace != "milvus" {
		t.Errorf("expected namespace milvus, got %s", spec.Global.Namespace)
	}

	// Check milvus defaults
	if spec.MilvusServers[0].Port != 19530 {
		t.Errorf("expected milvus port 19530, got %d", spec.MilvusServers[0].Port)
	}
	if spec.MilvusServers[0].Mode != ModeStandalone {
		t.Errorf("expected milvus mode standalone, got %s", spec.MilvusServers[0].Mode)
	}

	// Check etcd defaults
	if spec.EtcdServers[0].ClientPort != 2379 {
		t.Errorf("expected etcd client port 2379, got %d", spec.EtcdServers[0].ClientPort)
	}

	// Check minio defaults
	if spec.MinioServers[0].Port != 9000 {
		t.Errorf("expected minio port 9000, got %d", spec.MinioServers[0].Port)
	}
	if spec.MinioServers[0].AccessKey != "minioadmin" {
		t.Errorf("expected minio access key minioadmin, got %s", spec.MinioServers[0].AccessKey)
	}
}

func TestGetMode(t *testing.T) {
	tests := []struct {
		name     string
		spec     *Specification
		expected DeployMode
	}{
		{
			name:     "empty servers returns standalone",
			spec:     &Specification{},
			expected: ModeStandalone,
		},
		{
			name: "standalone mode",
			spec: &Specification{
				MilvusServers: []MilvusSpec{{Host: "localhost", Mode: ModeStandalone}},
			},
			expected: ModeStandalone,
		},
		{
			name: "distributed mode",
			spec: &Specification{
				MilvusServers: []MilvusSpec{{Host: "localhost", Mode: ModeDistributed}},
			},
			expected: ModeDistributed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.GetMode()
			if got != tt.expected {
				t.Errorf("GetMode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsDistributed(t *testing.T) {
	tests := []struct {
		name     string
		spec     *Specification
		expected bool
	}{
		{
			name: "standalone is not distributed",
			spec: &Specification{
				MilvusServers: []MilvusSpec{{Host: "localhost", Mode: ModeStandalone}},
			},
			expected: false,
		},
		{
			name: "distributed mode",
			spec: &Specification{
				MilvusServers: []MilvusSpec{{Host: "localhost", Mode: ModeDistributed}},
			},
			expected: true,
		},
		{
			name: "cluster mode (backward compat)",
			spec: &Specification{
				MilvusServers: []MilvusSpec{{Host: "localhost", Mode: ModeCluster}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.IsDistributed()
			if got != tt.expected {
				t.Errorf("IsDistributed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHasMonitoring(t *testing.T) {
	tests := []struct {
		name     string
		spec     *Specification
		expected bool
	}{
		{
			name:     "no monitoring",
			spec:     &Specification{},
			expected: false,
		},
		{
			name: "has monitor servers",
			spec: &Specification{
				MonitorServers: []MonitorSpec{{Host: "localhost"}},
			},
			expected: true,
		},
		{
			name: "has grafana servers",
			spec: &Specification{
				GrafanaServers: []GrafanaSpec{{Host: "localhost"}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.HasMonitoring()
			if got != tt.expected {
				t.Errorf("HasMonitoring() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetTLSMode(t *testing.T) {
	tests := []struct {
		name     string
		spec     *Specification
		expected int
	}{
		{
			name:     "default mode is 1",
			spec:     &Specification{},
			expected: 1,
		},
		{
			name: "explicit mode 2",
			spec: &Specification{
				Global: GlobalOptions{
					TLS: TLSConfig{Mode: 2},
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.spec.GetTLSMode()
			if got != tt.expected {
				t.Errorf("GetTLSMode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoadAndSaveSpecification(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "spec-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test specification
	spec := &Specification{
		Global: GlobalOptions{
			Namespace: "test-ns",
		},
		MilvusServers: []MilvusSpec{{Host: "milvus.local", Port: 19530}},
		EtcdServers:   []EtcdSpec{{Host: "etcd.local"}},
		MinioServers:  []MinioSpec{{Host: "minio.local"}},
	}

	// Save the specification
	specPath := filepath.Join(tmpDir, "topology.yaml")
	if err := SaveSpecification(spec, specPath); err != nil {
		t.Fatalf("failed to save specification: %v", err)
	}

	// Load the specification
	loaded, err := LoadSpecification(specPath)
	if err != nil {
		t.Fatalf("failed to load specification: %v", err)
	}

	// Verify loaded data
	if loaded.Global.Namespace != "test-ns" {
		t.Errorf("expected namespace test-ns, got %s", loaded.Global.Namespace)
	}
	if len(loaded.MilvusServers) != 1 || loaded.MilvusServers[0].Host != "milvus.local" {
		t.Errorf("milvus server not loaded correctly")
	}
}

func TestLoadSpecification_FileNotFound(t *testing.T) {
	_, err := LoadSpecification("/nonexistent/path/topology.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadSpecification_InvalidYAML(t *testing.T) {
	// Create a temporary file with invalid YAML
	tmpDir, err := os.MkdirTemp("", "spec-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	invalidPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidPath, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("failed to write invalid yaml: %v", err)
	}

	_, err = LoadSpecification(invalidPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
