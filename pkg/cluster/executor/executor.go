package executor

import (
	"context"
	"time"
)

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

	// Diagnose performs health diagnostics on the cluster
	Diagnose(ctx context.Context) (*DiagnoseResult, error)

	// Reload triggers a configuration reload
	// If config is provided, it merges the config before reloading
	// If wait is true, it waits for all pods to become ready
	Reload(ctx context.Context, opts ReloadOptions) error
}

// ReloadOptions defines options for reloading configuration
type ReloadOptions struct {
	// Config is the configuration to merge before reloading (optional)
	Config map[string]any

	// Wait indicates whether to wait for pods to become ready
	Wait bool

	// Timeout is the maximum time to wait for reload to complete
	Timeout time.Duration
}

// DiagnoseResult contains the results of a health diagnosis
type DiagnoseResult struct {
	// Overall health status
	Healthy bool `json:"healthy"`

	// Summary message
	Summary string `json:"summary"`

	// Component checks
	Components []ComponentCheck `json:"components"`

	// Connectivity checks
	Connectivity []ConnectivityCheck `json:"connectivity"`

	// Resource checks
	Resources []ResourceCheck `json:"resources"`

	// Issues found
	Issues []Issue `json:"issues"`
}

// CheckStatus represents the status of a check
type CheckStatus string

const (
	CheckStatusOK      CheckStatus = "OK"
	CheckStatusWarning CheckStatus = "WARNING"
	CheckStatusError   CheckStatus = "ERROR"
)

// ComponentCheck represents a component health check
type ComponentCheck struct {
	Name     string      `json:"name"`
	Status   CheckStatus `json:"status"`
	Message  string      `json:"message"`
	Replicas int         `json:"replicas,omitempty"`
	Ready    int         `json:"ready,omitempty"`
}

// ConnectivityCheck represents a connectivity check
type ConnectivityCheck struct {
	Name    string      `json:"name"`
	Target  string      `json:"target"`
	Status  CheckStatus `json:"status"`
	Latency string      `json:"latency,omitempty"`
	Message string      `json:"message"`
}

// ResourceCheck represents a resource usage check
type ResourceCheck struct {
	Name      string      `json:"name"`
	Status    CheckStatus `json:"status"`
	Usage     string      `json:"usage"`
	Limit     string      `json:"limit,omitempty"`
	Message   string      `json:"message"`
}

// Issue represents a diagnosed issue
type Issue struct {
	Severity    CheckStatus `json:"severity"`
	Component   string      `json:"component"`
	Description string      `json:"description"`
	Suggestion  string      `json:"suggestion"`
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
