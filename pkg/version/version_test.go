package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGetVersionInfo(t *testing.T) {
	info := GetVersionInfo()

	if info.Version != MiUpVersion {
		t.Errorf("Version = %s, want %s", info.Version, MiUpVersion)
	}
	if info.GitHash != GitHash {
		t.Errorf("GitHash = %s, want %s", info.GitHash, GitHash)
	}
	if info.GitBranch != GitBranch {
		t.Errorf("GitBranch = %s, want %s", info.GitBranch, GitBranch)
	}
	if info.BuildTime != BuildTime {
		t.Errorf("BuildTime = %s, want %s", info.BuildTime, BuildTime)
	}
	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %s, want %s", info.GoVersion, runtime.Version())
	}
	if info.OS != runtime.GOOS {
		t.Errorf("OS = %s, want %s", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %s, want %s", info.Arch, runtime.GOARCH)
	}
}

func TestInfo_String(t *testing.T) {
	info := GetVersionInfo()
	str := info.String()

	// Check that all fields are present
	expectedParts := []string{
		"MiUp",
		info.Version,
		"Git Commit:",
		info.GitHash,
		"Git Branch:",
		info.GitBranch,
		"Build Time:",
		info.BuildTime,
		"Go Version:",
		info.GoVersion,
		"OS/Arch:",
		info.OS,
		info.Arch,
	}

	for _, part := range expectedParts {
		if !strings.Contains(str, part) {
			t.Errorf("String() missing %q", part)
		}
	}
}

func TestInfo_ShortString(t *testing.T) {
	info := GetVersionInfo()
	short := info.ShortString()

	expected := "miup version " + info.Version
	if short != expected {
		t.Errorf("ShortString() = %q, want %q", short, expected)
	}
}

func TestDefaultValues(t *testing.T) {
	// Test default values are set
	if MiUpVersion == "" {
		t.Error("MiUpVersion should not be empty")
	}
	if GitHash == "" {
		t.Error("GitHash should not be empty")
	}
	if GitBranch == "" {
		t.Error("GitBranch should not be empty")
	}
	if BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}
}

func TestInfoStruct(t *testing.T) {
	info := Info{
		Version:   "v1.0.0",
		GitHash:   "abc123",
		GitBranch: "main",
		BuildTime: "2024-01-01",
		GoVersion: "go1.21",
		OS:        "linux",
		Arch:      "amd64",
	}

	if info.Version != "v1.0.0" {
		t.Errorf("Version = %s, want v1.0.0", info.Version)
	}

	str := info.String()
	if !strings.Contains(str, "v1.0.0") {
		t.Error("String() should contain version")
	}
	if !strings.Contains(str, "abc123") {
		t.Error("String() should contain git hash")
	}
	if !strings.Contains(str, "main") {
		t.Error("String() should contain git branch")
	}

	short := info.ShortString()
	if short != "miup version v1.0.0" {
		t.Errorf("ShortString() = %q, want %q", short, "miup version v1.0.0")
	}
}
