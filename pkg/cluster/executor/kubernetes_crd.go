package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/mmga-lab/miup/pkg/k8s"
	"gopkg.in/yaml.v3"
)

// KubernetesCRDOptions contains options for CRD-based deployment
type KubernetesCRDOptions struct {
	Kubeconfig  string
	Context     string
	Namespace   string
	ClusterName string
	CRDContent  []byte
}

// KubernetesCRDExecutor deploys Milvus directly from CRD YAML
type KubernetesCRDExecutor struct {
	client      *k8s.Client
	clusterName string
	namespace   string
	crdContent  []byte
}

// NewKubernetesCRDExecutor creates a new CRD-based executor
func NewKubernetesCRDExecutor(opts KubernetesCRDOptions) (*KubernetesCRDExecutor, error) {
	client, err := k8s.NewClient(k8s.ClientOptions{
		Kubeconfig: opts.Kubeconfig,
		Context:    opts.Context,
		Namespace:  opts.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &KubernetesCRDExecutor{
		client:      client,
		clusterName: opts.ClusterName,
		namespace:   opts.Namespace,
		crdContent:  opts.CRDContent,
	}, nil
}

// Deploy creates the Milvus cluster from CRD
func (e *KubernetesCRDExecutor) Deploy(ctx context.Context) error {
	// Check if Milvus Operator is installed
	installed, err := e.client.CheckMilvusOperatorInstalled(ctx)
	if err != nil {
		return fmt.Errorf("failed to check Milvus Operator: %w", err)
	}
	if !installed {
		return fmt.Errorf("Milvus Operator is not installed. Please install it first: kubectl apply -f https://raw.githubusercontent.com/zilliztech/milvus-operator/main/deploy/manifests/deployment.yaml")
	}

	// Parse the CRD YAML
	milvus, err := e.parseCRD()
	if err != nil {
		return fmt.Errorf("failed to parse CRD: %w", err)
	}

	// Override name and namespace
	milvus.Name = e.clusterName
	milvus.Namespace = e.namespace

	// Add managed-by label
	if milvus.Labels == nil {
		milvus.Labels = make(map[string]string)
	}
	milvus.Labels["app.kubernetes.io/managed-by"] = "miup"
	milvus.Labels["app.kubernetes.io/instance"] = e.clusterName

	// Create the Milvus resource
	if err := e.client.CreateMilvus(ctx, milvus); err != nil {
		return fmt.Errorf("failed to create Milvus: %w", err)
	}

	// Wait for ready
	return e.waitForReady(ctx, 10*time.Minute)
}

// parseCRD parses the CRD YAML content into a Milvus object
func (e *KubernetesCRDExecutor) parseCRD() (*k8s.Milvus, error) {
	// First parse into a generic map to extract TypeMeta fields
	var rawData map[string]any
	if err := yaml.Unmarshal(e.crdContent, &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CRD: %w", err)
	}

	// Validate Kind
	kind, _ := rawData["kind"].(string)
	if kind != "Milvus" {
		return nil, fmt.Errorf("invalid Kind: expected 'Milvus', got '%s'", kind)
	}

	// Parse the full structure
	var milvus k8s.Milvus
	if err := yaml.Unmarshal(e.crdContent, &milvus); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CRD: %w", err)
	}

	// Set TypeMeta explicitly (yaml.v3 doesn't handle inline well)
	milvus.APIVersion = "milvus.io/v1beta1"
	milvus.Kind = "Milvus"

	// Extract metadata from raw data
	if metadata, ok := rawData["metadata"].(map[string]any); ok {
		if name, ok := metadata["name"].(string); ok {
			milvus.Name = name
		}
		if namespace, ok := metadata["namespace"].(string); ok {
			milvus.Namespace = namespace
		}
		if labels, ok := metadata["labels"].(map[string]any); ok {
			milvus.Labels = make(map[string]string)
			for k, v := range labels {
				if vs, ok := v.(string); ok {
					milvus.Labels[k] = vs
				}
			}
		}
	}

	return &milvus, nil
}

// waitForReady waits for the Milvus cluster to become healthy
func (e *KubernetesCRDExecutor) waitForReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for cluster to be ready")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		if milvus.Status.Status == "Healthy" {
			return nil
		}

		time.Sleep(5 * time.Second)
	}
}

// Start starts the cluster
func (e *KubernetesCRDExecutor) Start(ctx context.Context) error {
	milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
	if err != nil {
		return err
	}

	// Restore replicas to 1 if it was set to 0
	if milvus.Spec.Components.Standalone != nil {
		one := int32(1)
		milvus.Spec.Components.Standalone.Replicas = &one
	}

	if err := e.client.UpdateMilvus(ctx, milvus); err != nil {
		return err
	}

	return e.waitForReady(ctx, 5*time.Minute)
}

// Stop stops the cluster
func (e *KubernetesCRDExecutor) Stop(ctx context.Context) error {
	milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
	if err != nil {
		return err
	}

	// Set replicas to 0 to stop the cluster
	if milvus.Spec.Components.Standalone != nil {
		zero := int32(0)
		milvus.Spec.Components.Standalone.Replicas = &zero
	}

	return e.client.UpdateMilvus(ctx, milvus)
}

// Destroy deletes the Milvus cluster
func (e *KubernetesCRDExecutor) Destroy(ctx context.Context) error {
	return e.client.DeleteMilvus(ctx, e.clusterName, e.namespace)
}

// Status returns the cluster status
func (e *KubernetesCRDExecutor) Status(ctx context.Context) (string, error) {
	milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
	if err != nil {
		return "", err
	}
	return milvus.Status.Status, nil
}

// IsRunning checks if the cluster is running
func (e *KubernetesCRDExecutor) IsRunning(ctx context.Context) (bool, error) {
	status, err := e.Status(ctx)
	if err != nil {
		return false, err
	}
	return status == "Healthy", nil
}

// Logs returns logs from the cluster
func (e *KubernetesCRDExecutor) Logs(ctx context.Context, service string, tail int) (string, error) {
	pods, err := e.client.GetMilvusPods(ctx, e.clusterName, e.namespace)
	if err != nil {
		return "", err
	}

	if len(pods) == 0 {
		return "", fmt.Errorf("no pods found for cluster")
	}

	var logs string
	for _, pod := range pods {
		podLogs, err := e.client.GetPodLogs(ctx, e.namespace, pod, "", int64(tail))
		if err != nil {
			continue
		}
		logs += fmt.Sprintf("--- %s ---\n%s\n", pod, podLogs)
	}

	return logs, nil
}

// Scale scales a component
func (e *KubernetesCRDExecutor) Scale(ctx context.Context, component string, opts ScaleOptions) error {
	// This would require updating the CRD directly
	// For now, delegate to the main executor logic
	return fmt.Errorf("scale not yet implemented for CRD executor")
}

// GetReplicas returns replica counts
func (e *KubernetesCRDExecutor) GetReplicas(ctx context.Context) (map[string]int, error) {
	milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
	if err != nil {
		return nil, err
	}

	replicas := make(map[string]int)

	if milvus.Spec.Mode == k8s.MilvusModeStandalone {
		if milvus.Spec.Components.Standalone != nil && milvus.Spec.Components.Standalone.Replicas != nil {
			replicas["standalone"] = int(*milvus.Spec.Components.Standalone.Replicas)
		} else {
			replicas["standalone"] = 1
		}
	}

	return replicas, nil
}

// Upgrade upgrades the cluster
func (e *KubernetesCRDExecutor) Upgrade(ctx context.Context, version string) error {
	milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
	if err != nil {
		return err
	}

	// Update the image
	if !hasVersionPrefix(version) {
		version = "v" + version
	}
	milvus.Spec.Components.Image = fmt.Sprintf("milvusdb/milvus:%s", version)

	if err := e.client.UpdateMilvus(ctx, milvus); err != nil {
		return err
	}

	return e.waitForReady(ctx, 15*time.Minute)
}

// GetVersion returns the current version
func (e *KubernetesCRDExecutor) GetVersion(ctx context.Context) (string, error) {
	milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
	if err != nil {
		return "", err
	}

	// Extract version from image
	image := milvus.Spec.Components.Image
	if image == "" {
		return "unknown", nil
	}

	// Parse version from image tag
	parts := splitLast(image, ":")
	if len(parts) == 2 {
		return parts[1], nil
	}

	return "unknown", nil
}

// GetConfig returns the cluster config
func (e *KubernetesCRDExecutor) GetConfig(ctx context.Context) (map[string]any, error) {
	milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
	if err != nil {
		return nil, err
	}
	return milvus.Spec.Config, nil
}

// SetConfig sets the cluster config
func (e *KubernetesCRDExecutor) SetConfig(ctx context.Context, config map[string]any) error {
	milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
	if err != nil {
		return err
	}

	if milvus.Spec.Config == nil {
		milvus.Spec.Config = make(map[string]any)
	}

	// Merge config
	for k, v := range config {
		milvus.Spec.Config[k] = v
	}

	return e.client.UpdateMilvus(ctx, milvus)
}

// Reload reloads the cluster configuration
func (e *KubernetesCRDExecutor) Reload(ctx context.Context, opts ReloadOptions) error {
	milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
	if err != nil {
		return err
	}

	// Add/update annotation to trigger reconciliation
	if milvus.Annotations == nil {
		milvus.Annotations = make(map[string]string)
	}
	milvus.Annotations["milvus.io/config-reload"] = time.Now().Format(time.RFC3339)

	if err := e.client.UpdateMilvus(ctx, milvus); err != nil {
		return err
	}

	if opts.Wait {
		timeout := opts.Timeout
		if timeout == 0 {
			timeout = 10 * time.Minute
		}
		return e.waitForReady(ctx, timeout)
	}

	return nil
}

// Diagnose performs health diagnostics
func (e *KubernetesCRDExecutor) Diagnose(ctx context.Context) (*DiagnoseResult, error) {
	result := &DiagnoseResult{
		Healthy: false,
	}

	milvus, err := e.client.GetMilvus(ctx, e.clusterName, e.namespace)
	if err != nil {
		result.Summary = fmt.Sprintf("Failed to get Milvus CRD: %v", err)
		result.Issues = append(result.Issues, Issue{
			Severity:    CheckStatusError,
			Component:   "milvus-crd",
			Description: fmt.Sprintf("Cannot access Milvus CRD: %v", err),
			Suggestion:  "Check if the cluster exists and verify kubeconfig is correct",
		})
		return result, nil
	}

	// Check status
	result.Healthy = milvus.Status.Status == "Healthy"
	if result.Healthy {
		result.Summary = "Cluster is healthy"
	} else {
		result.Summary = fmt.Sprintf("Cluster status: %s", milvus.Status.Status)
	}

	// Add component checks
	for name, status := range milvus.Status.ComponentsDeployStatus {
		checkStatus := CheckStatusOK
		if status.Status.ReadyReplicas != status.Status.Replicas {
			checkStatus = CheckStatusError
		}
		check := ComponentCheck{
			Name:     name,
			Status:   checkStatus,
			Message:  fmt.Sprintf("%d/%d replicas ready", status.Status.ReadyReplicas, status.Status.Replicas),
			Replicas: int(status.Status.Replicas),
			Ready:    int(status.Status.ReadyReplicas),
		}
		result.Components = append(result.Components, check)
	}

	// Add connectivity check
	result.Connectivity = append(result.Connectivity, ConnectivityCheck{
		Name:    "milvus-service",
		Target:  milvus.Status.Endpoint,
		Status:  CheckStatusOK,
		Message: "Service endpoint available",
	})

	return result, nil
}

// hasVersionPrefix checks if version has v prefix
func hasVersionPrefix(version string) bool {
	return len(version) > 0 && version[0] == 'v'
}

// splitLast splits string by last occurrence of separator
func splitLast(s, sep string) []string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == sep[0] {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
