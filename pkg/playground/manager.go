package playground

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zilliztech/miup/pkg/executor"
	"github.com/zilliztech/miup/pkg/localdata"
	"github.com/zilliztech/miup/pkg/logger"
)

const (
	// PlaygroundDir is the directory name for playground data
	PlaygroundDir = "playground"
	// MetaFileName is the metadata file name
	MetaFileName = "meta.json"
	// StartupTimeout is the timeout for waiting for services to start
	StartupTimeout = 5 * time.Minute
)

// Status represents the playground status
type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
	StatusUnknown Status = "unknown"
)

// Meta contains playground metadata
type Meta struct {
	Tag           string    `json:"tag"`
	Mode          Mode      `json:"mode"`
	MilvusVersion string    `json:"milvus_version"`
	WithMonitor   bool      `json:"with_monitor"`
	CreatedAt     time.Time `json:"created_at"`
	MilvusPort    int       `json:"milvus_port"`
	MinioPort     int       `json:"minio_port"`
}

// Manager manages playground instances
type Manager struct {
	profile *localdata.Profile
}

// NewManager creates a new playground manager
func NewManager(profile *localdata.Profile) *Manager {
	return &Manager{profile: profile}
}

// PlaygroundDir returns the path to a playground instance directory
func (m *Manager) PlaygroundDir(tag string) string {
	return m.profile.Path(PlaygroundDir, tag)
}

// MetaPath returns the path to the metadata file
func (m *Manager) MetaPath(tag string) string {
	return filepath.Join(m.PlaygroundDir(tag), MetaFileName)
}

// Start starts a new playground instance
func (m *Manager) Start(ctx context.Context, cfg *Config) error {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Check Docker availability
	if err := executor.CheckDockerAvailable(); err != nil {
		return err
	}
	if err := executor.CheckDockerComposeAvailable(); err != nil {
		return err
	}
	if err := executor.CheckDockerRunning(); err != nil {
		return err
	}

	// Check if playground already exists and is running
	if running, _ := m.IsRunning(ctx, cfg.Tag); running {
		return fmt.Errorf("playground '%s' is already running", cfg.Tag)
	}

	playgroundDir := m.PlaygroundDir(cfg.Tag)

	// Create playground directory
	if err := os.MkdirAll(playgroundDir, 0755); err != nil {
		return fmt.Errorf("failed to create playground directory: %w", err)
	}

	// Generate docker-compose.yaml
	composeContent, err := GenerateComposeFile(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate compose file: %w", err)
	}

	composePath := filepath.Join(playgroundDir, "docker-compose.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	// Generate prometheus config if monitoring is enabled
	if cfg.WithMonitor {
		prometheusConfig := GeneratePrometheusConfig(cfg)
		prometheusPath := filepath.Join(playgroundDir, "prometheus.yml")
		if err := os.WriteFile(prometheusPath, []byte(prometheusConfig), 0644); err != nil {
			return fmt.Errorf("failed to write prometheus config: %w", err)
		}
	}

	// Save metadata
	meta := &Meta{
		Tag:           cfg.Tag,
		Mode:          cfg.Mode,
		MilvusVersion: cfg.MilvusVersion,
		WithMonitor:   cfg.WithMonitor,
		CreatedAt:     time.Now(),
		MilvusPort:    cfg.MilvusPort,
		MinioPort:     cfg.MinioPort,
	}
	if err := m.saveMeta(cfg.Tag, meta); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	// Start docker compose
	logger.Info("Starting Milvus playground (mode: %s)...", cfg.Mode)
	compose := executor.NewDockerCompose(playgroundDir, fmt.Sprintf("miup-%s", cfg.Tag))

	if err := compose.Up(ctx); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	logger.Success("Playground '%s' started successfully!", cfg.Tag)
	return nil
}

// Stop stops a playground instance
func (m *Manager) Stop(ctx context.Context, tag string, removeVolumes bool) error {
	playgroundDir := m.PlaygroundDir(tag)

	if _, err := os.Stat(playgroundDir); os.IsNotExist(err) {
		return fmt.Errorf("playground '%s' does not exist", tag)
	}

	compose := executor.NewDockerCompose(playgroundDir, fmt.Sprintf("miup-%s", tag))

	if !compose.Exists() {
		return fmt.Errorf("playground '%s' is not properly configured", tag)
	}

	logger.Info("Stopping playground '%s'...", tag)
	if err := compose.Down(ctx, removeVolumes); err != nil {
		return fmt.Errorf("failed to stop services: %w", err)
	}

	logger.Success("Playground '%s' stopped!", tag)
	return nil
}

// Status returns the status of a playground instance
func (m *Manager) Status(ctx context.Context, tag string) (*InstanceStatus, error) {
	playgroundDir := m.PlaygroundDir(tag)

	if _, err := os.Stat(playgroundDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("playground '%s' does not exist", tag)
	}

	// Load metadata
	meta, err := m.loadMeta(tag)
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}

	compose := executor.NewDockerCompose(playgroundDir, fmt.Sprintf("miup-%s", tag))

	running, err := compose.IsRunning(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check status: %w", err)
	}

	status := StatusStopped
	if running {
		status = StatusRunning
	}

	// Get container status
	var containerStatus string
	if running {
		containerStatus, _ = compose.PS(ctx)
	}

	return &InstanceStatus{
		Meta:            meta,
		Status:          status,
		ContainerStatus: containerStatus,
	}, nil
}

// InstanceStatus contains the full status of a playground instance
type InstanceStatus struct {
	Meta            *Meta
	Status          Status
	ContainerStatus string
}

// IsRunning checks if a playground instance is running
func (m *Manager) IsRunning(ctx context.Context, tag string) (bool, error) {
	playgroundDir := m.PlaygroundDir(tag)

	if _, err := os.Stat(playgroundDir); os.IsNotExist(err) {
		return false, nil
	}

	compose := executor.NewDockerCompose(playgroundDir, fmt.Sprintf("miup-%s", tag))
	return compose.IsRunning(ctx)
}

// List lists all playground instances
func (m *Manager) List(ctx context.Context) ([]*InstanceStatus, error) {
	playgroundBaseDir := m.profile.Path(PlaygroundDir)

	entries, err := os.ReadDir(playgroundBaseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var instances []*InstanceStatus
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		status, err := m.Status(ctx, entry.Name())
		if err != nil {
			continue
		}

		instances = append(instances, status)
	}

	return instances, nil
}

// Logs retrieves logs from a playground instance
func (m *Manager) Logs(ctx context.Context, tag string, service string, tail int) (string, error) {
	playgroundDir := m.PlaygroundDir(tag)

	if _, err := os.Stat(playgroundDir); os.IsNotExist(err) {
		return "", fmt.Errorf("playground '%s' does not exist", tag)
	}

	compose := executor.NewDockerCompose(playgroundDir, fmt.Sprintf("miup-%s", tag))
	return compose.Logs(ctx, service, tail)
}

// Clean removes a playground instance completely
func (m *Manager) Clean(ctx context.Context, tag string) error {
	// First stop if running
	if running, _ := m.IsRunning(ctx, tag); running {
		if err := m.Stop(ctx, tag, true); err != nil {
			logger.Warn("Failed to stop playground: %v", err)
		}
	}

	playgroundDir := m.PlaygroundDir(tag)
	if err := os.RemoveAll(playgroundDir); err != nil {
		return fmt.Errorf("failed to remove playground directory: %w", err)
	}

	logger.Success("Playground '%s' cleaned up!", tag)
	return nil
}

func (m *Manager) saveMeta(tag string, meta *Meta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	metaPath := m.MetaPath(tag)
	return os.WriteFile(metaPath, data, 0644)
}

func (m *Manager) loadMeta(tag string) (*Meta, error) {
	metaPath := m.MetaPath(tag)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}
