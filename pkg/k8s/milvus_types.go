package k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// MilvusGroup is the API group for Milvus resources
	MilvusGroup = "milvus.io"
	// MilvusVersion is the API version for Milvus resources
	MilvusVersion = "v1beta1"
	// MilvusResource is the resource name for Milvus
	MilvusResource = "milvuses"
	// MilvusKind is the kind for Milvus resources
	MilvusKind = "Milvus"
)

// MilvusMode represents the deployment mode
type MilvusMode string

const (
	MilvusModeStandalone MilvusMode = "standalone"
	MilvusModeCluster    MilvusMode = "cluster"
)

// Milvus represents a Milvus cluster CRD
type Milvus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MilvusSpec   `json:"spec,omitempty"`
	Status MilvusStatus `json:"status,omitempty"`
}

// MilvusList is a list of Milvus resources
type MilvusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Milvus `json:"items"`
}

// MilvusSpec defines the desired state of Milvus
type MilvusSpec struct {
	// Mode specifies the deployment mode: standalone or cluster
	Mode MilvusMode `json:"mode,omitempty"`

	// Dependencies specifies the dependencies configuration
	Dependencies MilvusDependencies `json:"dependencies,omitempty"`

	// Components specifies the component configurations
	Components MilvusComponents `json:"components,omitempty"`

	// Config specifies custom configuration
	Config map[string]interface{} `json:"config,omitempty"`
}

// MilvusDependencies defines external dependencies
type MilvusDependencies struct {
	// Etcd configuration
	Etcd EtcdConfig `json:"etcd,omitempty"`

	// Storage configuration (MinIO/S3)
	Storage StorageConfig `json:"storage,omitempty"`

	// MsgStreamType specifies the message stream type
	MsgStreamType string `json:"msgStreamType,omitempty"`
}

// EtcdConfig defines etcd configuration
type EtcdConfig struct {
	// InCluster specifies in-cluster etcd configuration
	InCluster *InClusterConfig `json:"inCluster,omitempty"`

	// External specifies external etcd endpoints
	External *ExternalEtcdConfig `json:"external,omitempty"`
}

// InClusterConfig defines in-cluster component configuration
type InClusterConfig struct {
	// DeletionPolicy specifies deletion policy
	DeletionPolicy string `json:"deletionPolicy,omitempty"`

	// PVCDeletion specifies whether to delete PVC
	PVCDeletion bool `json:"pvcDeletion,omitempty"`

	// Values specifies helm values
	Values map[string]interface{} `json:"values,omitempty"`
}

// ExternalEtcdConfig defines external etcd configuration
type ExternalEtcdConfig struct {
	// Endpoints specifies etcd endpoints
	Endpoints []string `json:"endpoints,omitempty"`
}

// StorageConfig defines storage configuration
type StorageConfig struct {
	// InCluster specifies in-cluster MinIO configuration
	InCluster *InClusterConfig `json:"inCluster,omitempty"`

	// External specifies external S3-compatible storage
	External *ExternalStorageConfig `json:"external,omitempty"`
}

// ExternalStorageConfig defines external S3-compatible storage
type ExternalStorageConfig struct {
	// Endpoint specifies the S3 endpoint
	Endpoint string `json:"endpoint,omitempty"`

	// Bucket specifies the bucket name
	Bucket string `json:"bucket,omitempty"`

	// AccessKeyID specifies the access key
	AccessKeyID string `json:"accessKeyID,omitempty"`

	// SecretAccessKey specifies the secret key
	SecretAccessKey string `json:"secretAccessKey,omitempty"`

	// UseSSL specifies whether to use SSL
	UseSSL bool `json:"useSSL,omitempty"`

	// UseIAM specifies whether to use IAM role
	UseIAM bool `json:"useIAM,omitempty"`
}

// MilvusComponents defines component configurations
type MilvusComponents struct {
	// Image specifies the Milvus image
	Image string `json:"image,omitempty"`

	// ImagePullPolicy specifies the image pull policy
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets specifies image pull secrets
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`

	// DisableMetric disables metrics collection for all components
	DisableMetric bool `json:"disableMetric,omitempty"`

	// MetricInterval specifies the interval of podmonitor metric scraping
	MetricInterval string `json:"metricInterval,omitempty"`

	// Volumes specifies additional volumes to mount
	Volumes []Volume `json:"volumes,omitempty"`

	// VolumeMounts specifies additional volume mounts
	VolumeMounts []VolumeMount `json:"volumeMounts,omitempty"`

	// Standalone specifies standalone configuration
	Standalone *ComponentSpec `json:"standalone,omitempty"`

	// Proxy specifies proxy configuration
	Proxy *ComponentSpec `json:"proxy,omitempty"`

	// RootCoord specifies root coordinator configuration
	RootCoord *ComponentSpec `json:"rootCoord,omitempty"`

	// QueryCoord specifies query coordinator configuration
	QueryCoord *ComponentSpec `json:"queryCoord,omitempty"`

	// DataCoord specifies data coordinator configuration
	DataCoord *ComponentSpec `json:"dataCoord,omitempty"`

	// IndexCoord specifies index coordinator configuration
	IndexCoord *ComponentSpec `json:"indexCoord,omitempty"`

	// QueryNode specifies query node configuration
	QueryNode *ComponentSpec `json:"queryNode,omitempty"`

	// DataNode specifies data node configuration
	DataNode *ComponentSpec `json:"dataNode,omitempty"`

	// IndexNode specifies index node configuration
	IndexNode *ComponentSpec `json:"indexNode,omitempty"`
}

// ComponentSpec defines a component specification
type ComponentSpec struct {
	// Replicas specifies the number of replicas
	Replicas *int32 `json:"replicas,omitempty"`

	// Resources specifies resource requirements
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// NodeSelector specifies node selector
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations specifies tolerations
	Tolerations []interface{} `json:"tolerations,omitempty"`

	// Affinity specifies affinity rules
	Affinity interface{} `json:"affinity,omitempty"`
}

// ResourceRequirements defines resource requirements
type ResourceRequirements struct {
	Limits   map[string]string `json:"limits,omitempty"`
	Requests map[string]string `json:"requests,omitempty"`
}

// Volume defines a volume
type Volume struct {
	Name   string        `json:"name"`
	Secret *SecretSource `json:"secret,omitempty"`
}

// SecretSource defines a secret volume source
type SecretSource struct {
	SecretName string `json:"secretName"`
}

// VolumeMount defines a volume mount
type VolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

// MilvusStatus defines the observed state of Milvus
type MilvusStatus struct {
	// Status is the overall status
	Status string `json:"status,omitempty"`

	// Conditions are the status conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Endpoint is the service endpoint
	Endpoint string `json:"endpoint,omitempty"`

	// Replicas shows replica counts
	Replicas MilvusReplicas `json:"replicas,omitempty"`
}

// MilvusReplicas shows replica counts for components
type MilvusReplicas struct {
	Proxy     int32 `json:"proxy,omitempty"`
	RootCoord int32 `json:"rootCoord,omitempty"`
	DataCoord int32 `json:"dataCoord,omitempty"`
	DataNode  int32 `json:"dataNode,omitempty"`
	QueryNode int32 `json:"queryNode,omitempty"`
	IndexNode int32 `json:"indexNode,omitempty"`
}
