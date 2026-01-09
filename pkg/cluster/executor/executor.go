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

	// Scale scales a component to the specified number of replicas
	Scale(ctx context.Context, component string, replicas int) error

	// GetReplicas returns the current replica count for each component
	GetReplicas(ctx context.Context) (map[string]int, error)
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
