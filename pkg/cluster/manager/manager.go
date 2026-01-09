package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zilliztech/miup/pkg/cluster/executor"
	"github.com/zilliztech/miup/pkg/cluster/spec"
	"github.com/zilliztech/miup/pkg/localdata"
	"github.com/zilliztech/miup/pkg/logger"
)

const (
	// ClusterDir is the directory name for cluster data
	ClusterDir = "clusters"
	// MetaFileName is the metadata file name
	MetaFileName = "meta.json"
	// TopologyFileName is the topology file name
	TopologyFileName = "topology.yaml"
)

// Manager manages cluster lifecycle
type Manager struct {
	profile *localdata.Profile
}

// NewManager creates a new cluster manager
func NewManager(profile *localdata.Profile) *Manager {
	return &Manager{profile: profile}
}

// ClusterDir returns the path to a cluster directory
func (m *Manager) ClusterDir(name string) string {
	return m.profile.Path(ClusterDir, name)
}

// MetaPath returns the path to cluster metadata
func (m *Manager) MetaPath(name string) string {
	return filepath.Join(m.ClusterDir(name), MetaFileName)
}

// TopologyPath returns the path to cluster topology
func (m *Manager) TopologyPath(name string) string {
	return filepath.Join(m.ClusterDir(name), TopologyFileName)
}

// DeployOptions contains options for deployment
type DeployOptions struct {
	MilvusVersion string
	Backend       spec.BackendType
	SkipConfirm   bool

	// Kubernetes specific options
	Kubeconfig  string
	KubeContext string
	Namespace   string
	WithMonitor bool
}

// Deploy deploys a new cluster
func (m *Manager) Deploy(ctx context.Context, name string, topoPath string, opts DeployOptions) error {
	// Check if cluster already exists
	if m.Exists(name) {
		return fmt.Errorf("cluster '%s' already exists", name)
	}

	// Load and validate specification
	specification, err := spec.LoadSpecification(topoPath)
	if err != nil {
		return err
	}

	if err := specification.Validate(); err != nil {
		return fmt.Errorf("invalid topology: %w", err)
	}

	// Set default backend
	if opts.Backend == "" {
		opts.Backend = spec.BackendLocal
	}

	// Set default Milvus version
	if opts.MilvusVersion == "" {
		opts.MilvusVersion = "v2.5.4"
	}

	// Create cluster directory
	clusterDir := m.ClusterDir(name)
	if err := os.MkdirAll(clusterDir, 0755); err != nil {
		return fmt.Errorf("failed to create cluster directory: %w", err)
	}

	// Save topology
	if err := spec.SaveSpecification(specification, m.TopologyPath(name)); err != nil {
		return fmt.Errorf("failed to save topology: %w", err)
	}

	// Create and save metadata
	meta := spec.NewClusterMeta(name, specification, opts.Backend, opts.MilvusVersion)
	if err := spec.SaveMeta(meta, m.MetaPath(name)); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	// Create executor
	exec, err := m.createExecutor(name, specification, opts)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Deploy
	logger.Info("Deploying cluster '%s'...", name)
	if err := exec.Deploy(ctx); err != nil {
		meta.Status = spec.StatusUnknown
		spec.SaveMeta(meta, m.MetaPath(name))
		return fmt.Errorf("deployment failed: %w", err)
	}

	// Update status
	meta.Status = spec.StatusRunning
	if err := spec.SaveMeta(meta, m.MetaPath(name)); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	logger.Success("Cluster '%s' deployed successfully!", name)
	return nil
}

// Start starts a cluster
func (m *Manager) Start(ctx context.Context, name string) error {
	if !m.Exists(name) {
		return fmt.Errorf("cluster '%s' does not exist", name)
	}

	meta, err := spec.LoadMeta(m.MetaPath(name))
	if err != nil {
		return err
	}

	specification, err := spec.LoadSpecification(m.TopologyPath(name))
	if err != nil {
		return err
	}

	exec, err := m.createExecutor(name, specification, DeployOptions{
		Backend:       meta.Backend,
		MilvusVersion: meta.MilvusVersion,
	})
	if err != nil {
		return err
	}

	logger.Info("Starting cluster '%s'...", name)
	if err := exec.Start(ctx); err != nil {
		return fmt.Errorf("failed to start cluster: %w", err)
	}

	meta.Status = spec.StatusRunning
	if err := spec.SaveMeta(meta, m.MetaPath(name)); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	logger.Success("Cluster '%s' started!", name)
	return nil
}

// Stop stops a cluster
func (m *Manager) Stop(ctx context.Context, name string) error {
	if !m.Exists(name) {
		return fmt.Errorf("cluster '%s' does not exist", name)
	}

	meta, err := spec.LoadMeta(m.MetaPath(name))
	if err != nil {
		return err
	}

	specification, err := spec.LoadSpecification(m.TopologyPath(name))
	if err != nil {
		return err
	}

	exec, err := m.createExecutor(name, specification, DeployOptions{
		Backend:       meta.Backend,
		MilvusVersion: meta.MilvusVersion,
	})
	if err != nil {
		return err
	}

	logger.Info("Stopping cluster '%s'...", name)
	if err := exec.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop cluster: %w", err)
	}

	meta.Status = spec.StatusStopped
	if err := spec.SaveMeta(meta, m.MetaPath(name)); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	logger.Success("Cluster '%s' stopped!", name)
	return nil
}

// Destroy destroys a cluster
func (m *Manager) Destroy(ctx context.Context, name string, force bool) error {
	if !m.Exists(name) {
		return fmt.Errorf("cluster '%s' does not exist", name)
	}

	meta, err := spec.LoadMeta(m.MetaPath(name))
	if err != nil {
		return err
	}

	specification, err := spec.LoadSpecification(m.TopologyPath(name))
	if err != nil {
		return err
	}

	exec, err := m.createExecutor(name, specification, DeployOptions{
		Backend:       meta.Backend,
		MilvusVersion: meta.MilvusVersion,
	})
	if err != nil {
		return err
	}

	logger.Warn("Destroying cluster '%s'...", name)
	if err := exec.Destroy(ctx); err != nil {
		if !force {
			return fmt.Errorf("failed to destroy cluster: %w", err)
		}
		logger.Warn("Force destroying despite error: %v", err)
	}

	// Remove cluster directory
	if err := os.RemoveAll(m.ClusterDir(name)); err != nil {
		return fmt.Errorf("failed to remove cluster directory: %w", err)
	}

	logger.Success("Cluster '%s' destroyed!", name)
	return nil
}

// Display returns cluster information
func (m *Manager) Display(ctx context.Context, name string) (*ClusterInfo, error) {
	if !m.Exists(name) {
		return nil, fmt.Errorf("cluster '%s' does not exist", name)
	}

	meta, err := spec.LoadMeta(m.MetaPath(name))
	if err != nil {
		return nil, err
	}

	specification, err := spec.LoadSpecification(m.TopologyPath(name))
	if err != nil {
		return nil, err
	}

	exec, err := m.createExecutor(name, specification, DeployOptions{
		Backend:       meta.Backend,
		MilvusVersion: meta.MilvusVersion,
	})
	if err != nil {
		return nil, err
	}

	// Get container status
	containerStatus, _ := exec.Status(ctx)

	return &ClusterInfo{
		Meta:            meta,
		Spec:            specification,
		ContainerStatus: containerStatus,
	}, nil
}

// ClusterInfo contains complete cluster information
type ClusterInfo struct {
	Meta            *spec.ClusterMeta
	Spec            *spec.Specification
	ContainerStatus string
}

// List lists all clusters
func (m *Manager) List(ctx context.Context) ([]*spec.ClusterMeta, error) {
	clustersDir := m.profile.Path(ClusterDir)

	entries, err := os.ReadDir(clustersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var clusters []*spec.ClusterMeta
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		meta, err := spec.LoadMeta(m.MetaPath(entry.Name()))
		if err != nil {
			continue
		}

		// Check actual status
		specification, err := spec.LoadSpecification(m.TopologyPath(entry.Name()))
		if err == nil {
			exec, err := m.createExecutor(entry.Name(), specification, DeployOptions{
				Backend:       meta.Backend,
				MilvusVersion: meta.MilvusVersion,
			})
			if err == nil {
				if running, _ := exec.IsRunning(ctx); running {
					meta.Status = spec.StatusRunning
				} else {
					meta.Status = spec.StatusStopped
				}
			}
		}

		clusters = append(clusters, meta)
	}

	return clusters, nil
}

// Logs retrieves logs from a cluster
func (m *Manager) Logs(ctx context.Context, name string, service string, tail int) (string, error) {
	if !m.Exists(name) {
		return "", fmt.Errorf("cluster '%s' does not exist", name)
	}

	meta, err := spec.LoadMeta(m.MetaPath(name))
	if err != nil {
		return "", err
	}

	specification, err := spec.LoadSpecification(m.TopologyPath(name))
	if err != nil {
		return "", err
	}

	exec, err := m.createExecutor(name, specification, DeployOptions{
		Backend:       meta.Backend,
		MilvusVersion: meta.MilvusVersion,
	})
	if err != nil {
		return "", err
	}

	return exec.Logs(ctx, service, tail)
}

// Scale scales a component in the cluster
func (m *Manager) Scale(ctx context.Context, name string, component string, replicas int) error {
	if !m.Exists(name) {
		return fmt.Errorf("cluster '%s' does not exist", name)
	}

	meta, err := spec.LoadMeta(m.MetaPath(name))
	if err != nil {
		return err
	}

	specification, err := spec.LoadSpecification(m.TopologyPath(name))
	if err != nil {
		return err
	}

	exec, err := m.createExecutor(name, specification, DeployOptions{
		Backend:       meta.Backend,
		MilvusVersion: meta.MilvusVersion,
	})
	if err != nil {
		return err
	}

	// Update status to scaling
	oldStatus := meta.Status
	meta.Status = spec.StatusScaling
	if err := spec.SaveMeta(meta, m.MetaPath(name)); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	logger.Info("Scaling %s to %d replicas in cluster '%s'...", component, replicas, name)
	if err := exec.Scale(ctx, component, replicas); err != nil {
		// Restore old status on failure
		meta.Status = oldStatus
		spec.SaveMeta(meta, m.MetaPath(name))
		return fmt.Errorf("failed to scale: %w", err)
	}

	// Update status back to running
	meta.Status = spec.StatusRunning
	if err := spec.SaveMeta(meta, m.MetaPath(name)); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	logger.Success("Scaled %s to %d replicas!", component, replicas)
	return nil
}

// GetReplicas returns the current replica count for each component
func (m *Manager) GetReplicas(ctx context.Context, name string) (map[string]int, error) {
	if !m.Exists(name) {
		return nil, fmt.Errorf("cluster '%s' does not exist", name)
	}

	meta, err := spec.LoadMeta(m.MetaPath(name))
	if err != nil {
		return nil, err
	}

	specification, err := spec.LoadSpecification(m.TopologyPath(name))
	if err != nil {
		return nil, err
	}

	exec, err := m.createExecutor(name, specification, DeployOptions{
		Backend:       meta.Backend,
		MilvusVersion: meta.MilvusVersion,
	})
	if err != nil {
		return nil, err
	}

	return exec.GetReplicas(ctx)
}

// Exists checks if a cluster exists
func (m *Manager) Exists(name string) bool {
	_, err := os.Stat(m.ClusterDir(name))
	return err == nil
}

// createExecutor creates the appropriate executor based on backend
func (m *Manager) createExecutor(name string, specification *spec.Specification, opts DeployOptions) (executor.Executor, error) {
	switch opts.Backend {
	case spec.BackendLocal:
		return executor.NewLocalExecutor(m.ClusterDir(name), name, specification, opts.MilvusVersion)
	case spec.BackendKubernetes:
		namespace := opts.Namespace
		if namespace == "" {
			namespace = specification.Global.Namespace
		}
		return executor.NewKubernetesExecutor(executor.KubernetesOptions{
			Kubeconfig:    opts.Kubeconfig,
			Context:       opts.KubeContext,
			Namespace:     namespace,
			ClusterName:   name,
			Spec:          specification,
			MilvusVersion: opts.MilvusVersion,
			WithMonitor:   opts.WithMonitor,
		})
	default:
		return nil, fmt.Errorf("unknown backend: %s", opts.Backend)
	}
}
