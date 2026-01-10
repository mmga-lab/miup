package component

import (
	"testing"
)

func TestSupportsPlatform(t *testing.T) {
	comp := Registry["birdwatcher"]
	if comp == nil {
		t.Fatal("birdwatcher should be in registry")
	}

	tests := []struct {
		os       string
		arch     string
		expected bool
	}{
		{"darwin", "amd64", true},
		{"darwin", "arm64", true},
		{"linux", "amd64", true},
		{"linux", "arm64", true},
		{"windows", "amd64", false},
		{"darwin", "386", false},
		{"linux", "mips", false},
		{"freebsd", "amd64", false},
	}

	for _, tt := range tests {
		t.Run(tt.os+"/"+tt.arch, func(t *testing.T) {
			got := comp.SupportsPlatform(tt.os, tt.arch)
			if got != tt.expected {
				t.Errorf("SupportsPlatform(%s, %s) = %v, want %v", tt.os, tt.arch, got, tt.expected)
			}
		})
	}
}

func TestGetComponent(t *testing.T) {
	t.Run("existing component", func(t *testing.T) {
		comp, ok := GetComponent("birdwatcher")
		if !ok {
			t.Fatal("GetComponent(birdwatcher) should return true")
		}
		if comp == nil {
			t.Fatal("GetComponent(birdwatcher) should return non-nil component")
		}
		if comp.Name != "birdwatcher" {
			t.Errorf("Name = %s, want birdwatcher", comp.Name)
		}
	})

	t.Run("non-existing component", func(t *testing.T) {
		comp, ok := GetComponent("nonexistent")
		if ok {
			t.Error("GetComponent(nonexistent) should return false")
		}
		if comp != nil {
			t.Error("GetComponent(nonexistent) should return nil")
		}
	})
}

func TestListAvailable(t *testing.T) {
	components := ListAvailable()
	if len(components) == 0 {
		t.Error("ListAvailable() should return at least one component")
	}

	// Check that all registered components are in the list
	names := make(map[string]bool)
	for _, comp := range components {
		names[comp.Name] = true
	}

	for name := range Registry {
		if !names[name] {
			t.Errorf("Component %s not in ListAvailable() result", name)
		}
	}
}

func TestRegistry(t *testing.T) {
	// Test that expected components are registered
	expectedComponents := []string{"birdwatcher", "milvus-backup"}

	for _, name := range expectedComponents {
		comp, ok := Registry[name]
		if !ok {
			t.Errorf("Component %s should be in registry", name)
			continue
		}
		if comp.Name != name {
			t.Errorf("Component name = %s, want %s", comp.Name, name)
		}
		if comp.Description == "" {
			t.Errorf("Component %s should have description", name)
		}
		if comp.Repo == "" {
			t.Errorf("Component %s should have repo", name)
		}
		if comp.Binary == "" {
			t.Errorf("Component %s should have binary", name)
		}
		if comp.AssetName == nil {
			t.Errorf("Component %s should have AssetName function", name)
		}
	}
}

func TestBirdwatcherAssetName(t *testing.T) {
	comp := Registry["birdwatcher"]
	if comp == nil {
		t.Fatal("birdwatcher should be in registry")
	}

	tests := []struct {
		version  string
		os       string
		arch     string
		expected string
	}{
		{"v0.1.0", "darwin", "arm64", "birdwatcher_Darwin_arm64.tar.gz"},
		{"v0.1.0", "darwin", "amd64", "birdwatcher_Darwin_x86_64.tar.gz"},
		{"v0.1.0", "linux", "arm64", "birdwatcher_Linux_arm64.tar.gz"},
		{"v0.1.0", "linux", "amd64", "birdwatcher_Linux_x86_64.tar.gz"},
	}

	for _, tt := range tests {
		t.Run(tt.os+"/"+tt.arch, func(t *testing.T) {
			got := comp.AssetName(tt.version, tt.os, tt.arch)
			if got != tt.expected {
				t.Errorf("AssetName(%s, %s, %s) = %s, want %s", tt.version, tt.os, tt.arch, got, tt.expected)
			}
		})
	}
}

func TestMilvusBackupAssetName(t *testing.T) {
	comp := Registry["milvus-backup"]
	if comp == nil {
		t.Fatal("milvus-backup should be in registry")
	}

	tests := []struct {
		version  string
		os       string
		arch     string
		expected string
	}{
		{"v0.5.9", "darwin", "arm64", "milvus-backup_0.5.9_Darwin_arm64.tar.gz"},
		{"v0.5.9", "darwin", "amd64", "milvus-backup_0.5.9_Darwin_x86_64.tar.gz"},
		{"v0.5.9", "linux", "arm64", "milvus-backup_0.5.9_Linux_arm64.tar.gz"},
		{"v0.5.9", "linux", "amd64", "milvus-backup_0.5.9_Linux_x86_64.tar.gz"},
		{"0.5.9", "linux", "amd64", "milvus-backup_0.5.9_Linux_x86_64.tar.gz"}, // No v prefix
	}

	for _, tt := range tests {
		t.Run(tt.version+"_"+tt.os+"/"+tt.arch, func(t *testing.T) {
			got := comp.AssetName(tt.version, tt.os, tt.arch)
			if got != tt.expected {
				t.Errorf("AssetName(%s, %s, %s) = %s, want %s", tt.version, tt.os, tt.arch, got, tt.expected)
			}
		})
	}
}

func TestCapitalizeOS(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"darwin", "Darwin"},
		{"linux", "Linux"},
		{"windows", "windows"}, // Unknown OS passes through
		{"freebsd", "freebsd"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := capitalizeOS(tt.input)
			if got != tt.expected {
				t.Errorf("capitalizeOS(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizeArch(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"amd64", "x86_64"},
		{"arm64", "arm64"},
		{"386", "386"},    // Unknown arch passes through
		{"mips", "mips"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeArch(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeArch(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCurrentPlatformAssetName(t *testing.T) {
	comp := Registry["birdwatcher"]
	if comp == nil {
		t.Fatal("birdwatcher should be in registry")
	}

	// Just test that it returns a non-empty string
	assetName := comp.CurrentPlatformAssetName("v1.0.0")
	if assetName == "" {
		t.Error("CurrentPlatformAssetName should return non-empty string")
	}
	if len(assetName) < 10 {
		t.Errorf("CurrentPlatformAssetName returned too short: %s", assetName)
	}
}
