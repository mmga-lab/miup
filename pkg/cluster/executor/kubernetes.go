package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zilliztech/miup/pkg/cluster/spec"
	"github.com/zilliztech/miup/pkg/k8s"
)

// KubernetesExecutor executes cluster operations on Kubernetes using Milvus Operator
type KubernetesExecutor struct {
	client        *k8s.Client
	clusterName   string
	namespace     string
	spec          *spec.Specification
	milvusVersion string
	withMonitor   bool
}

// KubernetesOptions contains options for creating a Kubernetes executor
type KubernetesOptions struct {
	Kubeconfig    string
	Context       string
	Namespace     string
	ClusterName   string
	Spec          *spec.Specification
	MilvusVersion string
	WithMonitor   bool
}

// NewKubernetesExecutor creates a new Kubernetes executor
func NewKubernetesExecutor(opts KubernetesOptions) (*KubernetesExecutor, error) {
	client, err := k8s.NewClient(k8s.ClientOptions{
		Kubeconfig: opts.Kubeconfig,
		Context:    opts.Context,
		Namespace:  opts.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	namespace := opts.Namespace
	if namespace == "" {
		namespace = client.Namespace()
	}

	return &KubernetesExecutor{
		client:        client,
		clusterName:   opts.ClusterName,
		namespace:     namespace,
		spec:          opts.Spec,
		milvusVersion: opts.MilvusVersion,
		withMonitor:   opts.WithMonitor,
	}, nil
}

// Deploy deploys the Milvus cluster using Milvus Operator
func (e *KubernetesExecutor) Deploy(ctx context.Context) error {
	// Check if Milvus Operator is installed
	installed, err := e.client.CheckMilvusOperatorInstalled(ctx)
	if err != nil {
		return fmt.Errorf("failed to check Milvus Operator: %w", err)
	}
	if !installed {
		return fmt.Errorf("Milvus Operator is not installed. Please install it first:\n" +
			"  kubectl apply -f https://raw.githubusercontent.com/zilliztech/milvus-operator/main/deploy/manifests/deployment.yaml")
	}

	// Convert spec to Milvus CRD
	milvus := e.specToMilvus()

	// Create the Milvus resource
	if err := e.client.CreateMilvus(ctx, milvus); err != nil {
		return fmt.Errorf("failed to create Milvus cluster: %w", err)
	}

	// Wait for the cluster to be ready
	return e.waitForReady(ctx, 10*time.Minute)
}

// Start is a no-op for Kubernetes (Operator manages state)
func (e *KubernetesExecutor) Start(ctx context.Context) error {
	// Check current status
	milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
	if err != nil {
		return fmt.Errorf("failed to get Milvus cluster: %w", err)
	}

	if milvus.Status.Status == "Healthy" {
		return nil // Already running
	}

	// For Kubernetes, we can scale up replicas if they were scaled to 0
	// For now, just wait for healthy status
	return e.waitForReady(ctx, 5*time.Minute)
}

// Stop scales down the Milvus cluster (set replicas to 0)
func (e *KubernetesExecutor) Stop(ctx context.Context) error {
	milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
	if err != nil {
		return fmt.Errorf("failed to get Milvus cluster: %w", err)
	}

	// Scale down all components to 0
	zero := int32(0)
	if milvus.Spec.Mode == k8s.MilvusModeStandalone {
		if milvus.Spec.Components.Standalone == nil {
			milvus.Spec.Components.Standalone = &k8s.ComponentSpec{}
		}
		milvus.Spec.Components.Standalone.Replicas = &zero
	} else {
		if milvus.Spec.Components.Proxy == nil {
			milvus.Spec.Components.Proxy = &k8s.ComponentSpec{}
		}
		milvus.Spec.Components.Proxy.Replicas = &zero

		if milvus.Spec.Components.QueryNode == nil {
			milvus.Spec.Components.QueryNode = &k8s.ComponentSpec{}
		}
		milvus.Spec.Components.QueryNode.Replicas = &zero

		if milvus.Spec.Components.DataNode == nil {
			milvus.Spec.Components.DataNode = &k8s.ComponentSpec{}
		}
		milvus.Spec.Components.DataNode.Replicas = &zero

		if milvus.Spec.Components.IndexNode == nil {
			milvus.Spec.Components.IndexNode = &k8s.ComponentSpec{}
		}
		milvus.Spec.Components.IndexNode.Replicas = &zero
	}

	return e.client.UpdateMilvus(ctx, milvus)
}

// Destroy deletes the Milvus cluster
func (e *KubernetesExecutor) Destroy(ctx context.Context) error {
	return e.client.DeleteMilvus(ctx, e.clusterName, e.namespace)
}

// Status returns the cluster status
func (e *KubernetesExecutor) Status(ctx context.Context) (string, error) {
	milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
	if err != nil {
		return "", fmt.Errorf("failed to get Milvus cluster: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Name:      %s\n", milvus.Name))
	sb.WriteString(fmt.Sprintf("Namespace: %s\n", milvus.Namespace))
	sb.WriteString(fmt.Sprintf("Status:    %s\n", milvus.Status.Status))
	sb.WriteString(fmt.Sprintf("Endpoint:  %s\n", milvus.Status.Endpoint))

	// Show conditions
	if len(milvus.Status.Conditions) > 0 {
		sb.WriteString("\nConditions:\n")
		for _, cond := range milvus.Status.Conditions {
			sb.WriteString(fmt.Sprintf("  - %s: %s (%s)\n", cond.Type, cond.Status, cond.Message))
		}
	}

	// Show replicas
	sb.WriteString("\nReplicas:\n")
	if milvus.Spec.Mode == k8s.MilvusModeStandalone {
		sb.WriteString("  Standalone: 1\n")
	} else {
		sb.WriteString(fmt.Sprintf("  Proxy:     %d\n", milvus.Status.Replicas.Proxy))
		sb.WriteString(fmt.Sprintf("  QueryNode: %d\n", milvus.Status.Replicas.QueryNode))
		sb.WriteString(fmt.Sprintf("  DataNode:  %d\n", milvus.Status.Replicas.DataNode))
		sb.WriteString(fmt.Sprintf("  IndexNode: %d\n", milvus.Status.Replicas.IndexNode))
	}

	return sb.String(), nil
}

// IsRunning checks if the cluster is running
func (e *KubernetesExecutor) IsRunning(ctx context.Context) (bool, error) {
	milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
	if err != nil {
		return false, nil
	}
	return milvus.Status.Status == "Healthy", nil
}

// Logs retrieves logs from a service
func (e *KubernetesExecutor) Logs(ctx context.Context, service string, tail int) (string, error) {
	pods, err := e.client.GetMilvusPods(ctx, e.clusterName, e.namespace)
	if err != nil {
		return "", fmt.Errorf("failed to get pods: %w", err)
	}

	if len(pods) == 0 {
		return "", fmt.Errorf("no pods found for cluster %s", e.clusterName)
	}

	var sb strings.Builder
	for _, pod := range pods {
		// Filter by service if specified
		if service != "" && !strings.Contains(pod, service) {
			continue
		}

		logs, err := e.client.GetPodLogs(ctx, e.namespace, pod, "", int64(tail))
		if err != nil {
			sb.WriteString(fmt.Sprintf("--- %s (error: %v) ---\n", pod, err))
			continue
		}

		sb.WriteString(fmt.Sprintf("--- %s ---\n%s\n", pod, logs))
	}

	return sb.String(), nil
}

// waitForReady waits for the cluster to become healthy
func (e *KubernetesExecutor) waitForReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		if milvus.Status.Status == "Healthy" {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}

	return fmt.Errorf("timeout waiting for cluster to become healthy")
}

// specToMilvus converts the specification to a Milvus CRD
func (e *KubernetesExecutor) specToMilvus() *k8s.Milvus {
	mode := k8s.MilvusModeStandalone
	if e.spec.IsDistributed() {
		mode = k8s.MilvusModeCluster
	}

	milvus := &k8s.Milvus{
		Spec: k8s.MilvusSpec{
			Mode: mode,
			Dependencies: k8s.MilvusDependencies{
				Etcd:    e.buildEtcdConfig(),
				Storage: e.buildStorageConfig(),
			},
			Components: e.buildComponents(),
		},
	}

	// Set metadata
	milvus.Name = e.clusterName
	milvus.Namespace = e.namespace
	milvus.Labels = map[string]string{
		"app":                          "milvus",
		"app.kubernetes.io/name":       "milvus",
		"app.kubernetes.io/instance":   e.clusterName,
		"app.kubernetes.io/managed-by": "miup",
	}

	// Set image version
	if e.milvusVersion != "" {
		milvus.Spec.Components.Image = fmt.Sprintf("milvusdb/milvus:%s", e.milvusVersion)
	}

	// Configure monitoring (enabled by default, Milvus Operator creates PodMonitor)
	// DisableMetric=false enables metrics, MetricInterval sets scrape interval
	if e.withMonitor {
		milvus.Spec.Components.DisableMetric = false
		milvus.Spec.Components.MetricInterval = "15s"
	}

	// Configure TLS if enabled
	if e.spec.HasTLS() {
		e.configureTLS(milvus)
	}

	return milvus
}

// configureTLS configures TLS for the Milvus CRD
func (e *KubernetesExecutor) configureTLS(milvus *k8s.Milvus) {
	tlsConfig := e.spec.Global.TLS
	tlsMode := e.spec.GetTLSMode()

	// Mount TLS secret as volume
	secretName := tlsConfig.SecretName
	if secretName == "" {
		secretName = fmt.Sprintf("%s-tls", e.clusterName)
	}

	milvus.Spec.Components.Volumes = []k8s.Volume{
		{
			Name: "tls-certs",
			Secret: &k8s.SecretSource{
				SecretName: secretName,
			},
		},
	}

	milvus.Spec.Components.VolumeMounts = []k8s.VolumeMount{
		{
			Name:      "tls-certs",
			MountPath: "/milvus/tls",
			ReadOnly:  true,
		},
	}

	// Set TLS configuration in milvus config
	if milvus.Spec.Config == nil {
		milvus.Spec.Config = make(map[string]interface{})
	}

	// TLS paths
	milvus.Spec.Config["tls"] = map[string]interface{}{
		"serverPemPath": "/milvus/tls/server.pem",
		"serverKeyPath": "/milvus/tls/server.key",
		"caPemPath":     "/milvus/tls/ca.pem",
	}

	// TLS mode in common.security
	milvus.Spec.Config["common"] = map[string]interface{}{
		"security": map[string]interface{}{
			"tlsMode": tlsMode,
		},
	}

	// Internal TLS if enabled
	if tlsConfig.InternalEnabled {
		common := milvus.Spec.Config["common"].(map[string]interface{})
		security := common["security"].(map[string]interface{})
		security["internaltlsEnabled"] = true

		milvus.Spec.Config["internaltls"] = map[string]interface{}{
			"serverPemPath": "/milvus/tls/server.pem",
			"serverKeyPath": "/milvus/tls/server.key",
			"caPemPath":     "/milvus/tls/ca.pem",
		}
	}
}

// buildEtcdConfig builds etcd configuration
func (e *KubernetesExecutor) buildEtcdConfig() k8s.EtcdConfig {
	// Check if external etcd is configured
	if len(e.spec.EtcdServers) > 0 && e.spec.EtcdServers[0].Host != "127.0.0.1" && e.spec.EtcdServers[0].Host != "localhost" {
		endpoints := make([]string, 0, len(e.spec.EtcdServers))
		for _, etcd := range e.spec.EtcdServers {
			endpoints = append(endpoints, fmt.Sprintf("%s:%d", etcd.Host, etcd.ClientPort))
		}
		return k8s.EtcdConfig{
			External: &k8s.ExternalEtcdConfig{
				Endpoints: endpoints,
			},
		}
	}

	// Use in-cluster etcd
	replicaCount := 3
	if e.spec.GetMode() == spec.ModeStandalone {
		replicaCount = 1
	}

	return k8s.EtcdConfig{
		InCluster: &k8s.InClusterConfig{
			DeletionPolicy: "Delete",
			PVCDeletion:    true,
			Values: map[string]interface{}{
				"replicaCount": replicaCount,
			},
		},
	}
}

// buildStorageConfig builds storage configuration
func (e *KubernetesExecutor) buildStorageConfig() k8s.StorageConfig {
	// Check if external MinIO/S3 is configured
	if len(e.spec.MinioServers) > 0 && e.spec.MinioServers[0].Host != "127.0.0.1" && e.spec.MinioServers[0].Host != "localhost" {
		minio := e.spec.MinioServers[0]
		return k8s.StorageConfig{
			External: &k8s.ExternalStorageConfig{
				Endpoint:        fmt.Sprintf("%s:%d", minio.Host, minio.Port),
				Bucket:          "milvus",
				AccessKeyID:     minio.AccessKey,
				SecretAccessKey: minio.SecretKey,
			},
		}
	}

	// Use in-cluster MinIO
	storageMode := "standalone"
	if e.spec.IsDistributed() {
		storageMode = "distributed"
	}

	return k8s.StorageConfig{
		InCluster: &k8s.InClusterConfig{
			DeletionPolicy: "Delete",
			PVCDeletion:    true,
			Values: map[string]interface{}{
				"mode": storageMode,
				"resources": map[string]interface{}{
					"requests": map[string]string{
						"memory": "256Mi",
					},
				},
			},
		},
	}
}

// buildComponents builds component configuration
func (e *KubernetesExecutor) buildComponents() k8s.MilvusComponents {
	components := k8s.MilvusComponents{}

	if e.spec.GetMode() == spec.ModeStandalone {
		one := int32(1)
		components.Standalone = &k8s.ComponentSpec{
			Replicas: &one,
		}
	} else {
		// Cluster mode - get replicas from spec (defaults are already set)
		milvusSpec := e.spec.MilvusServers[0]

		proxyReplicas := int32(milvusSpec.Components.Proxy.Replicas)
		components.Proxy = &k8s.ComponentSpec{Replicas: &proxyReplicas}

		rootCoordReplicas := int32(milvusSpec.Components.RootCoord.Replicas)
		components.RootCoord = &k8s.ComponentSpec{Replicas: &rootCoordReplicas}

		queryCoordReplicas := int32(milvusSpec.Components.QueryCoord.Replicas)
		components.QueryCoord = &k8s.ComponentSpec{Replicas: &queryCoordReplicas}

		dataCoordReplicas := int32(milvusSpec.Components.DataCoord.Replicas)
		components.DataCoord = &k8s.ComponentSpec{Replicas: &dataCoordReplicas}

		indexCoordReplicas := int32(milvusSpec.Components.IndexCoord.Replicas)
		components.IndexCoord = &k8s.ComponentSpec{Replicas: &indexCoordReplicas}

		queryNodeReplicas := int32(milvusSpec.Components.QueryNode.Replicas)
		components.QueryNode = &k8s.ComponentSpec{Replicas: &queryNodeReplicas}

		dataNodeReplicas := int32(milvusSpec.Components.DataNode.Replicas)
		components.DataNode = &k8s.ComponentSpec{Replicas: &dataNodeReplicas}

		indexNodeReplicas := int32(milvusSpec.Components.IndexNode.Replicas)
		components.IndexNode = &k8s.ComponentSpec{Replicas: &indexNodeReplicas}
	}

	return components
}

// GetEndpoint returns the Milvus service endpoint
func (e *KubernetesExecutor) GetEndpoint(ctx context.Context) (string, error) {
	return e.client.GetMilvusService(ctx, e.clusterName, e.namespace)
}
