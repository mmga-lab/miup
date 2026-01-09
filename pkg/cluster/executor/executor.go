package executor

import "context"

// Executor defines the interface for cluster execution backends
type Executor interface {
	// Deploy deploys the cluster
	Deploy(ctx context.Context) error

	// Start starts the cluster
	Start(ctx context.Context) error

	// Stop stops the cluster
	Stop(ctx context.Context) error

	// Destroy destroys the cluster and removes all data
	Destroy(ctx context.Context) error

	// Status returns the cluster status
	Status(ctx context.Context) (string, error)

	// IsRunning checks if the cluster is running
	IsRunning(ctx context.Context) (bool, error)

	// Logs retrieves logs from a service
	Logs(ctx context.Context, service string, tail int) (string, error)

	// Scale scales a component with the specified options
	Scale(ctx context.Context, component string, opts ScaleOptions) error

	// GetReplicas returns the current replica count for each component
	GetReplicas(ctx context.Context) (map[string]int, error)

	// Upgrade upgrades Milvus to the specified version
	Upgrade(ctx context.Context, version string) error

	// GetVersion returns the current Milvus version
	GetVersion(ctx context.Context) (string, error)

	// GetConfig returns the current Milvus configuration
	GetConfig(ctx context.Context) (map[string]interface{}, error)

	// SetConfig updates the Milvus configuration
	SetConfig(ctx context.Context, config map[string]interface{}) error
}

// ScaleOptions defines options for scaling a component
type ScaleOptions struct {
	// Replicas is the target number of replicas (0 means no change)
	Replicas int

	// CPURequest is the CPU request (e.g., "2", "500m")
	CPURequest string

	// CPULimit is the CPU limit (e.g., "4", "1000m")
	CPULimit string

	// MemoryRequest is the memory request (e.g., "4Gi", "512Mi")
	MemoryRequest string

	// MemoryLimit is the memory limit (e.g., "8Gi", "1024Mi")
	MemoryLimit string
}

// HasReplicaChange returns true if replicas should be changed
func (o ScaleOptions) HasReplicaChange() bool {
	return o.Replicas > 0
}

// HasResourceChange returns true if any resource should be changed
func (o ScaleOptions) HasResourceChange() bool {
	return o.CPURequest != "" || o.CPULimit != "" || o.MemoryRequest != "" || o.MemoryLimit != ""
}

// ComponentNames defines valid component names for scaling
var ComponentNames = []string{
	"proxy",
	"querynode",
	"datanode",
	"indexnode",
	"rootcoord",
	"querycoord",
	"datacoord",
	"indexcoord",
	"standalone",
}
