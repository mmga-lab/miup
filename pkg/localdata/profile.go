package localdata

import (
	"os"
	"path/filepath"
)

const (
	// ProfileDirName is the name of the profile directory
	ProfileDirName = ".miup"
	// ComponentParentDir is the directory to store components
	ComponentParentDir = "components"
	// DataParentDir is the directory to store cluster data
	DataParentDir = "data"
	// StorageParentDir is the directory to store cluster metadata
	StorageParentDir = "storage"
	// TelemetryDir is the directory for telemetry data
	TelemetryDir = "telemetry"
)

// Profile represents a local profile for miup
type Profile struct {
	root string
}

// NewProfile creates a new profile with the given root directory
func NewProfile(root string) *Profile {
	return &Profile{root: root}
}

// DefaultProfile returns the default profile based on MIUP_HOME or HOME
func DefaultProfile() (*Profile, error) {
	root := os.Getenv("MIUP_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		root = filepath.Join(home, ProfileDirName)
	}
	return NewProfile(root), nil
}

// Root returns the root directory of the profile
func (p *Profile) Root() string {
	return p.root
}

// Path returns the full path of a relative path within the profile
func (p *Profile) Path(relPath ...string) string {
	return filepath.Join(append([]string{p.root}, relPath...)...)
}

// ComponentsDir returns the components directory path
func (p *Profile) ComponentsDir() string {
	return p.Path(ComponentParentDir)
}

// ComponentDir returns a specific component directory path
func (p *Profile) ComponentDir(component string) string {
	return p.Path(ComponentParentDir, component)
}

// DataDir returns the data directory path
func (p *Profile) DataDir() string {
	return p.Path(DataParentDir)
}

// ClusterDataDir returns a specific cluster data directory path
func (p *Profile) ClusterDataDir(cluster string) string {
	return p.Path(DataParentDir, cluster)
}

// StorageDir returns the storage directory path
func (p *Profile) StorageDir() string {
	return p.Path(StorageParentDir)
}

// ClusterMetaPath returns the path to a cluster's metadata file
func (p *Profile) ClusterMetaPath(cluster string) string {
	return p.Path(StorageParentDir, cluster, "meta.yaml")
}

// ClusterTopologyPath returns the path to a cluster's topology file
func (p *Profile) ClusterTopologyPath(cluster string) string {
	return p.Path(StorageParentDir, cluster, "topology.yaml")
}

// EnsureDir ensures the directory exists
func (p *Profile) EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// InitProfile initializes the profile directory structure
func (p *Profile) InitProfile() error {
	dirs := []string{
		p.ComponentsDir(),
		p.DataDir(),
		p.StorageDir(),
		p.Path(TelemetryDir),
	}

	for _, dir := range dirs {
		if err := p.EnsureDir(dir); err != nil {
			return err
		}
	}
	return nil
}

// Exists checks if the profile exists
func (p *Profile) Exists() bool {
	_, err := os.Stat(p.root)
	return err == nil
}
