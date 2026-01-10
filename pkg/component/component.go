package component

import (
	"fmt"
	"runtime"
	"strings"
)

// Component represents an installable Milvus ecosystem tool
type Component struct {
	Name        string // e.g., "birdwatcher"
	Description string // Brief description
	Repo        string // GitHub repo, e.g., "milvus-io/birdwatcher"
	Binary      string // Binary name after extraction
}

// ComponentDef defines a component with its asset naming function
type ComponentDef struct {
	Component
	// AssetName returns the asset filename for a given version and platform
	AssetName func(version, os, arch string) string
}

// SupportsPlatform checks if the component supports the given OS/Arch
func (c *ComponentDef) SupportsPlatform(os, arch string) bool {
	// Currently all supported components work on darwin/linux with amd64/arm64
	switch os {
	case "darwin", "linux":
		switch arch {
		case "amd64", "arm64":
			return true
		}
	}
	return false
}

// Registry holds all supported components
var Registry = map[string]*ComponentDef{
	"birdwatcher": {
		Component: Component{
			Name:        "birdwatcher",
			Description: "Milvus diagnostic and debugging tool",
			Repo:        "milvus-io/birdwatcher",
			Binary:      "birdwatcher",
		},
		// Asset pattern: birdwatcher_Darwin_arm64.tar.gz
		AssetName: func(version, os, arch string) string {
			osName := capitalizeOS(os)
			archName := normalizeArch(arch)
			return fmt.Sprintf("birdwatcher_%s_%s.tar.gz", osName, archName)
		},
	},
	"milvus-backup": {
		Component: Component{
			Name:        "milvus-backup",
			Description: "Milvus backup and restore utility",
			Repo:        "zilliztech/milvus-backup",
			Binary:      "milvus-backup",
		},
		// Asset pattern: milvus-backup_0.5.9_Darwin_arm64.tar.gz
		AssetName: func(version, os, arch string) string {
			ver := strings.TrimPrefix(version, "v")
			osName := capitalizeOS(os)
			archName := normalizeArch(arch)
			return fmt.Sprintf("milvus-backup_%s_%s_%s.tar.gz", ver, osName, archName)
		},
	},
}

// GetComponent returns a component definition by name
func GetComponent(name string) (*ComponentDef, bool) {
	comp, ok := Registry[name]
	return comp, ok
}

// ListAvailable returns all available component definitions
func ListAvailable() []*ComponentDef {
	var components []*ComponentDef
	for _, comp := range Registry {
		components = append(components, comp)
	}
	return components
}

// CurrentPlatformAssetName returns the asset name for the current platform
func (c *ComponentDef) CurrentPlatformAssetName(version string) string {
	return c.AssetName(version, runtime.GOOS, runtime.GOARCH)
}

// capitalizeOS converts os name to title case for asset naming
func capitalizeOS(os string) string {
	switch os {
	case "darwin":
		return "Darwin"
	case "linux":
		return "Linux"
	default:
		return os
	}
}

// normalizeArch converts arch name for asset naming
func normalizeArch(arch string) string {
	switch arch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "arm64"
	default:
		return arch
	}
}
