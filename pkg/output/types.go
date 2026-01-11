package output

import "time"

// VersionInfo represents version information.
type VersionInfo struct {
	Version   string `json:"version"`
	GitHash   string `json:"git_hash,omitempty"`
	GitBranch string `json:"git_branch,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
	GoVersion string `json:"go_version,omitempty"`
	OS        string `json:"os,omitempty"`
	Arch      string `json:"arch,omitempty"`
}

// ComponentInfo represents information about an installed component.
type ComponentInfo struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Active      bool      `json:"active"`
	InstalledAt time.Time `json:"installed_at"`
	Path        string    `json:"path"`
}

// AvailableComponent represents an available (not installed) component.
type AvailableComponent struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Repo        string `json:"repo"`
}

// ComponentList represents a list of components.
type ComponentList struct {
	Components []ComponentInfo `json:"components"`
}

// InstanceSummary represents summary information about a Milvus instance.
type InstanceSummary struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Mode      string    `json:"mode"`
	Backend   string    `json:"backend"`
	Version   string    `json:"version"`
	Port      int       `json:"port"`
	Namespace string    `json:"namespace,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// InstanceList represents a list of instances.
type InstanceList struct {
	Instances []InstanceSummary `json:"instances"`
}

// InstanceInfo represents detailed information about a Milvus instance.
type InstanceInfo struct {
	Name      string                 `json:"name"`
	Status    string                 `json:"status"`
	Mode      string                 `json:"mode"`
	Backend   string                 `json:"backend"`
	Version   string                 `json:"version"`
	Port      int                    `json:"port"`
	Namespace string                 `json:"namespace,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	Config    map[string]interface{} `json:"config,omitempty"`
	Replicas  map[string]int         `json:"replicas,omitempty"`
}

// ServiceStatus represents the status of a service.
type ServiceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Ready  int    `json:"ready"`
	Total  int    `json:"total"`
}

// PlaygroundSummary represents summary information about a playground.
type PlaygroundSummary struct {
	Tag       string    `json:"tag"`
	Status    string    `json:"status"`
	Mode      string    `json:"mode"`
	Version   string    `json:"version"`
	Port      int       `json:"port"`
	CreatedAt time.Time `json:"created_at"`
}

// PlaygroundList represents a list of playgrounds.
type PlaygroundList struct {
	Playgrounds []PlaygroundSummary `json:"playgrounds"`
}

// PlaygroundStatus represents detailed status of a playground.
type PlaygroundStatus struct {
	Tag       string          `json:"tag"`
	Status    string          `json:"status"`
	Mode      string          `json:"mode"`
	Version   string          `json:"version"`
	Port      int             `json:"port"`
	CreatedAt time.Time       `json:"created_at"`
	Services  []ServiceStatus `json:"services,omitempty"`
}
