package component

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// InstalledVersion represents an installed version of a component
type InstalledVersion struct {
	Version     string    `json:"version"`
	InstalledAt time.Time `json:"installed_at"`
	BinaryPath  string    `json:"binary_path"`
	AssetName   string    `json:"asset_name"`
}

// ComponentMeta contains metadata for an installed component
type ComponentMeta struct {
	Name      string                       `json:"name"`
	Versions  map[string]*InstalledVersion `json:"versions"`
	Active    string                       `json:"active"` // Currently active version
	UpdatedAt time.Time                    `json:"updated_at"`
}

// MetaFileName is the metadata filename for each component
const MetaFileName = "meta.json"

// SaveMeta saves component metadata to the specified path
func SaveMeta(meta *ComponentMeta, path string) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}
	return nil
}

// LoadMeta loads component metadata from the specified path
func LoadMeta(path string) (*ComponentMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}
	var meta ComponentMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}
	return &meta, nil
}
