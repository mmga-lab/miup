package localdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewProfile(t *testing.T) {
	root := "/test/path"
	p := NewProfile(root)
	if p.Root() != root {
		t.Errorf("Root() = %s, want %s", p.Root(), root)
	}
}

func TestProfile_Path(t *testing.T) {
	p := NewProfile("/root")
	tests := []struct {
		name     string
		relPath  []string
		expected string
	}{
		{"single element", []string{"foo"}, "/root/foo"},
		{"multiple elements", []string{"foo", "bar"}, "/root/foo/bar"},
		{"empty", []string{}, "/root"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Path(tt.relPath...)
			if got != tt.expected {
				t.Errorf("Path(%v) = %s, want %s", tt.relPath, got, tt.expected)
			}
		})
	}
}

func TestProfile_DirectoryPaths(t *testing.T) {
	p := NewProfile("/root")

	tests := []struct {
		name     string
		method   func() string
		expected string
	}{
		{"ComponentsDir", p.ComponentsDir, "/root/components"},
		{"DataDir", p.DataDir, "/root/data"},
		{"StorageDir", p.StorageDir, "/root/storage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.method()
			if got != tt.expected {
				t.Errorf("%s() = %s, want %s", tt.name, got, tt.expected)
			}
		})
	}
}

func TestProfile_ComponentDir(t *testing.T) {
	p := NewProfile("/root")
	got := p.ComponentDir("milvus")
	expected := "/root/components/milvus"
	if got != expected {
		t.Errorf("ComponentDir(milvus) = %s, want %s", got, expected)
	}
}

func TestProfile_ClusterDataDir(t *testing.T) {
	p := NewProfile("/root")
	got := p.ClusterDataDir("my-cluster")
	expected := "/root/data/my-cluster"
	if got != expected {
		t.Errorf("ClusterDataDir(my-cluster) = %s, want %s", got, expected)
	}
}

func TestProfile_ClusterMetaPath(t *testing.T) {
	p := NewProfile("/root")
	got := p.ClusterMetaPath("my-cluster")
	expected := "/root/storage/my-cluster/meta.yaml"
	if got != expected {
		t.Errorf("ClusterMetaPath(my-cluster) = %s, want %s", got, expected)
	}
}

func TestProfile_ClusterTopologyPath(t *testing.T) {
	p := NewProfile("/root")
	got := p.ClusterTopologyPath("my-cluster")
	expected := "/root/storage/my-cluster/topology.yaml"
	if got != expected {
		t.Errorf("ClusterTopologyPath(my-cluster) = %s, want %s", got, expected)
	}
}

func TestProfile_InitProfile(t *testing.T) {
	tmpDir := t.TempDir()
	p := NewProfile(tmpDir)

	if err := p.InitProfile(); err != nil {
		t.Fatalf("InitProfile() error = %v", err)
	}

	// Check all directories exist
	expectedDirs := []string{
		p.ComponentsDir(),
		p.DataDir(),
		p.StorageDir(),
		p.Path(TelemetryDir),
	}

	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}
}

func TestProfile_EnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	p := NewProfile(tmpDir)

	newDir := filepath.Join(tmpDir, "new", "nested", "dir")
	if err := p.EnsureDir(newDir); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}

	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Errorf("Directory %s was not created", newDir)
	}

	// Should not error if directory already exists
	if err := p.EnsureDir(newDir); err != nil {
		t.Errorf("EnsureDir() on existing dir error = %v", err)
	}
}

func TestProfile_Exists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		p := NewProfile(tmpDir)
		if !p.Exists() {
			t.Error("Exists() = false, want true for existing directory")
		}
	})

	t.Run("not exists", func(t *testing.T) {
		p := NewProfile("/nonexistent/path/that/should/not/exist")
		if p.Exists() {
			t.Error("Exists() = true, want false for non-existing directory")
		}
	})
}

func TestDefaultProfile(t *testing.T) {
	// Test with MIUP_HOME set
	t.Run("with MIUP_HOME", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("MIUP_HOME", tmpDir)

		p, err := DefaultProfile()
		if err != nil {
			t.Fatalf("DefaultProfile() error = %v", err)
		}
		if p.Root() != tmpDir {
			t.Errorf("Root() = %s, want %s", p.Root(), tmpDir)
		}
	})

	// Test without MIUP_HOME (uses home dir)
	t.Run("without MIUP_HOME", func(t *testing.T) {
		t.Setenv("MIUP_HOME", "")

		p, err := DefaultProfile()
		if err != nil {
			t.Fatalf("DefaultProfile() error = %v", err)
		}

		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ProfileDirName)
		if p.Root() != expected {
			t.Errorf("Root() = %s, want %s", p.Root(), expected)
		}
	})
}

func TestConstants(t *testing.T) {
	if ProfileDirName != ".miup" {
		t.Errorf("ProfileDirName = %s, want .miup", ProfileDirName)
	}
	if ComponentParentDir != "components" {
		t.Errorf("ComponentParentDir = %s, want components", ComponentParentDir)
	}
	if DataParentDir != "data" {
		t.Errorf("DataParentDir = %s, want data", DataParentDir)
	}
	if StorageParentDir != "storage" {
		t.Errorf("StorageParentDir = %s, want storage", StorageParentDir)
	}
	if TelemetryDir != "telemetry" {
		t.Errorf("TelemetryDir = %s, want telemetry", TelemetryDir)
	}
}
