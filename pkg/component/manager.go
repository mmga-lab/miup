package component

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mmga-lab/miup/pkg/localdata"
	"github.com/mmga-lab/miup/pkg/logger"
)

// Manager manages component installation and execution
type Manager struct {
	profile    *localdata.Profile
	downloader *Downloader
}

// NewManager creates a new component manager
func NewManager(profile *localdata.Profile) *Manager {
	return &Manager{
		profile:    profile,
		downloader: NewDownloader(),
	}
}

// Install installs a component at the specified version
func (m *Manager) Install(ctx context.Context, name, version string) error {
	// Look up component in registry
	compDef, ok := Registry[name]
	if !ok {
		return fmt.Errorf("unknown component: %s (available: birdwatcher, milvus-backup)", name)
	}

	// Validate platform support
	if !compDef.SupportsPlatform(runtime.GOOS, runtime.GOARCH) {
		return fmt.Errorf("component %s does not support %s/%s", name, runtime.GOOS, runtime.GOARCH)
	}

	// Get release info
	var release *GitHubRelease
	var err error
	if version == "" || version == "latest" {
		logger.Info("Fetching latest release for %s...", name)
		release, err = m.downloader.GetLatestRelease(ctx, compDef.Repo)
	} else {
		// Normalize version
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}
		logger.Info("Fetching release %s for %s...", version, name)
		release, err = m.downloader.GetRelease(ctx, compDef.Repo, version)
	}
	if err != nil {
		return fmt.Errorf("failed to get release: %w", err)
	}

	version = release.TagName
	logger.Info("Installing %s %s...", name, version)

	// Check if already installed
	versionDir := m.VersionDir(name, version)
	existing := false
	if _, err := os.Stat(versionDir); err == nil {
		logger.Warn("Version %s is already installed, reinstalling...", version)
		existing = true
	}

	// Find matching asset
	asset, err := FindAsset(release, compDef.AssetName)
	if err != nil {
		return err
	}

	// Download and extract
	downloadDir := versionDir
	tempDir := ""
	if existing {
		var err error
		tempDir, err = os.MkdirTemp(m.ComponentDir(name), version+".tmp-")
		if err != nil {
			return fmt.Errorf("failed to create temp dir: %w", err)
		}
		downloadDir = tempDir
	}
	if err := m.downloader.DownloadAsset(ctx, asset, downloadDir); err != nil {
		if tempDir != "" {
			os.RemoveAll(tempDir)
		} else {
			os.RemoveAll(versionDir)
		}
		return fmt.Errorf("failed to download: %w", err)
	}
	if existing {
		backupDir := versionDir + ".bak"
		os.RemoveAll(backupDir)
		if err := os.Rename(versionDir, backupDir); err != nil {
			os.RemoveAll(tempDir)
			return fmt.Errorf("failed to backup existing version: %w", err)
		}
		if err := os.Rename(tempDir, versionDir); err != nil {
			_ = os.Rename(backupDir, versionDir)
			os.RemoveAll(tempDir)
			return fmt.Errorf("failed to replace existing version: %w", err)
		}
		os.RemoveAll(backupDir)
	}

	// Make binary executable
	binaryPath := m.BinaryPath(name, version)
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permission: %w", err)
	}

	// Update metadata
	if err := m.updateMeta(name, version, asset.Name); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	logger.Success("Installed %s %s", name, version)
	logger.Info("Binary: %s", binaryPath)

	return nil
}

// Uninstall removes a component version
func (m *Manager) Uninstall(ctx context.Context, name, version string) error {
	compDir := m.ComponentDir(name)

	// Check if component exists
	if _, err := os.Stat(compDir); os.IsNotExist(err) {
		return fmt.Errorf("component %s is not installed", name)
	}

	if version == "" {
		// Uninstall all versions
		if err := os.RemoveAll(compDir); err != nil {
			return fmt.Errorf("failed to remove component: %w", err)
		}
		logger.Success("Uninstalled all versions of %s", name)
	} else {
		// Normalize version
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}
		versionDir := m.VersionDir(name, version)
		if _, err := os.Stat(versionDir); os.IsNotExist(err) {
			return fmt.Errorf("version %s of %s is not installed", version, name)
		}
		if err := os.RemoveAll(versionDir); err != nil {
			return fmt.Errorf("failed to remove version: %w", err)
		}

		// Update metadata
		metaPath := filepath.Join(compDir, MetaFileName)
		meta, _ := LoadMeta(metaPath)
		if meta != nil {
			delete(meta.Versions, version)
			if meta.Active == version {
				// Set new active version
				meta.Active = ""
				for v := range meta.Versions {
					meta.Active = v
					break
				}
			}
			meta.UpdatedAt = time.Now()
			SaveMeta(meta, metaPath)
		}

		logger.Success("Uninstalled %s %s", name, version)
	}
	return nil
}

// List returns all installed components
func (m *Manager) List(ctx context.Context) ([]*ComponentMeta, error) {
	componentsDir := m.profile.ComponentsDir()
	entries, err := os.ReadDir(componentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var components []*ComponentMeta
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(m.ComponentDir(entry.Name()), MetaFileName)
		meta, err := LoadMeta(metaPath)
		if err != nil || meta == nil {
			continue
		}
		components = append(components, meta)
	}
	return components, nil
}

// Run executes an installed component
func (m *Manager) Run(ctx context.Context, name, version string, args []string) error {
	// Look up component
	if _, ok := Registry[name]; !ok {
		return fmt.Errorf("unknown component: %s", name)
	}

	if version == "" {
		// Find active/latest version
		meta, err := LoadMeta(filepath.Join(m.ComponentDir(name), MetaFileName))
		if err != nil {
			return fmt.Errorf("failed to load component metadata: %w", err)
		}
		if meta == nil {
			return fmt.Errorf("component %s is not installed", name)
		}
		version = meta.Active
		if version == "" {
			return fmt.Errorf("no active version for %s", name)
		}
	} else {
		// Normalize version
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}
	}

	binaryPath := m.BinaryPath(name, version)
	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("binary not found: %s (is %s %s installed?)", binaryPath, name, version)
	}

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ComponentDir returns the directory for a component
func (m *Manager) ComponentDir(name string) string {
	return m.profile.ComponentDir(name)
}

// VersionDir returns the directory for a specific version of a component
func (m *Manager) VersionDir(name, version string) string {
	return filepath.Join(m.ComponentDir(name), version)
}

// BinaryPath returns the path to the binary for a specific version
func (m *Manager) BinaryPath(name, version string) string {
	compDef := Registry[name]
	if compDef == nil {
		return ""
	}
	return filepath.Join(m.VersionDir(name, version), compDef.Binary)
}

func (m *Manager) updateMeta(name, version, assetName string) error {
	compDir := m.ComponentDir(name)
	if err := os.MkdirAll(compDir, 0755); err != nil {
		return err
	}

	metaPath := filepath.Join(compDir, MetaFileName)
	meta, _ := LoadMeta(metaPath)
	if meta == nil {
		meta = &ComponentMeta{
			Name:     name,
			Versions: make(map[string]*InstalledVersion),
		}
	}

	meta.Versions[version] = &InstalledVersion{
		Version:     version,
		InstalledAt: time.Now(),
		BinaryPath:  m.BinaryPath(name, version),
		AssetName:   assetName,
	}
	meta.Active = version
	meta.UpdatedAt = time.Now()

	return SaveMeta(meta, metaPath)
}
