package component

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetaFileName(t *testing.T) {
	if MetaFileName != "meta.json" {
		t.Errorf("MetaFileName = %s, want meta.json", MetaFileName)
	}
}

func TestSaveAndLoadMeta(t *testing.T) {
	tmpDir := t.TempDir()
	metaPath := filepath.Join(tmpDir, MetaFileName)

	now := time.Now().Truncate(time.Second) // Truncate for JSON precision

	meta := &ComponentMeta{
		Name:   "test-component",
		Active: "v1.0.0",
		Versions: map[string]*InstalledVersion{
			"v1.0.0": {
				Version:     "v1.0.0",
				InstalledAt: now,
				BinaryPath:  "/path/to/binary",
				AssetName:   "test_Darwin_arm64.tar.gz",
			},
		},
		UpdatedAt: now,
	}

	// Save
	if err := SaveMeta(meta, metaPath); err != nil {
		t.Fatalf("SaveMeta() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Fatal("SaveMeta() did not create file")
	}

	// Load
	loaded, err := LoadMeta(metaPath)
	if err != nil {
		t.Fatalf("LoadMeta() error = %v", err)
	}

	// Verify
	if loaded.Name != meta.Name {
		t.Errorf("Name = %s, want %s", loaded.Name, meta.Name)
	}
	if loaded.Active != meta.Active {
		t.Errorf("Active = %s, want %s", loaded.Active, meta.Active)
	}
	if len(loaded.Versions) != len(meta.Versions) {
		t.Errorf("Versions count = %d, want %d", len(loaded.Versions), len(meta.Versions))
	}

	v := loaded.Versions["v1.0.0"]
	if v == nil {
		t.Fatal("Version v1.0.0 not found")
	}
	if v.Version != "v1.0.0" {
		t.Errorf("Version.Version = %s, want v1.0.0", v.Version)
	}
	if v.BinaryPath != "/path/to/binary" {
		t.Errorf("Version.BinaryPath = %s, want /path/to/binary", v.BinaryPath)
	}
	if v.AssetName != "test_Darwin_arm64.tar.gz" {
		t.Errorf("Version.AssetName = %s, want test_Darwin_arm64.tar.gz", v.AssetName)
	}
}

func TestLoadMeta_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	metaPath := filepath.Join(tmpDir, "nonexistent.json")

	meta, err := LoadMeta(metaPath)
	if err != nil {
		t.Errorf("LoadMeta() for non-existent file error = %v, want nil", err)
	}
	if meta != nil {
		t.Error("LoadMeta() for non-existent file should return nil")
	}
}

func TestLoadMeta_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	metaPath := filepath.Join(tmpDir, MetaFileName)

	// Write invalid JSON
	if err := os.WriteFile(metaPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	meta, err := LoadMeta(metaPath)
	if err == nil {
		t.Error("LoadMeta() should return error for invalid JSON")
	}
	if meta != nil {
		t.Error("LoadMeta() should return nil for invalid JSON")
	}
}

func TestSaveMeta_InvalidPath(t *testing.T) {
	meta := &ComponentMeta{
		Name:     "test",
		Versions: make(map[string]*InstalledVersion),
	}

	// Try to save to an invalid path
	err := SaveMeta(meta, "/nonexistent/path/to/meta.json")
	if err == nil {
		t.Error("SaveMeta() should return error for invalid path")
	}
}

func TestComponentMeta_MultipleVersions(t *testing.T) {
	tmpDir := t.TempDir()
	metaPath := filepath.Join(tmpDir, MetaFileName)

	now := time.Now()

	meta := &ComponentMeta{
		Name:   "multi-version",
		Active: "v2.0.0",
		Versions: map[string]*InstalledVersion{
			"v1.0.0": {
				Version:     "v1.0.0",
				InstalledAt: now.Add(-24 * time.Hour),
				BinaryPath:  "/path/v1",
				AssetName:   "asset_v1.tar.gz",
			},
			"v2.0.0": {
				Version:     "v2.0.0",
				InstalledAt: now,
				BinaryPath:  "/path/v2",
				AssetName:   "asset_v2.tar.gz",
			},
		},
		UpdatedAt: now,
	}

	if err := SaveMeta(meta, metaPath); err != nil {
		t.Fatalf("SaveMeta() error = %v", err)
	}

	loaded, err := LoadMeta(metaPath)
	if err != nil {
		t.Fatalf("LoadMeta() error = %v", err)
	}

	if len(loaded.Versions) != 2 {
		t.Errorf("Versions count = %d, want 2", len(loaded.Versions))
	}
	if loaded.Active != "v2.0.0" {
		t.Errorf("Active = %s, want v2.0.0", loaded.Active)
	}
}

func TestInstalledVersionStruct(t *testing.T) {
	now := time.Now()
	v := &InstalledVersion{
		Version:     "v1.0.0",
		InstalledAt: now,
		BinaryPath:  "/usr/local/bin/tool",
		AssetName:   "tool_Linux_x86_64.tar.gz",
	}

	if v.Version != "v1.0.0" {
		t.Errorf("Version = %s, want v1.0.0", v.Version)
	}
	if v.InstalledAt != now {
		t.Errorf("InstalledAt mismatch")
	}
	if v.BinaryPath != "/usr/local/bin/tool" {
		t.Errorf("BinaryPath = %s, want /usr/local/bin/tool", v.BinaryPath)
	}
	if v.AssetName != "tool_Linux_x86_64.tar.gz" {
		t.Errorf("AssetName = %s, want tool_Linux_x86_64.tar.gz", v.AssetName)
	}
}
