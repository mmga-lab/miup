package check

import (
	"context"
	"fmt"
	"os"
	"strings"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/mmga-lab/miup/pkg/k8s"
)

// Status represents the status of a check
type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

// Result represents the result of a single check
type Result struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message"`
	Suggest string `json:"suggest,omitempty"`
}

// Report represents the complete check report
type Report struct {
	Results  []Result `json:"results"`
	Summary  Summary  `json:"summary"`
	CanDeploy bool    `json:"can_deploy"`
}

// Summary contains check statistics
type Summary struct {
	Total  int `json:"total"`
	Passed int `json:"passed"`
	Warned int `json:"warned"`
	Failed int `json:"failed"`
}

// Options contains options for the checker
type Options struct {
	Kubeconfig   string
	Context      string
	Namespace    string
	StorageClass string
}

// Checker performs environment checks
type Checker struct {
	opts      Options
	config    *rest.Config
	clientset *kubernetes.Clientset
}

// NewChecker creates a new checker
func NewChecker(opts Options) (*Checker, error) {
	config, err := buildConfig(opts.Kubeconfig, opts.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Checker{
		opts:      opts,
		config:    config,
		clientset: clientset,
	}, nil
}

// Run runs all checks and returns a report
func (c *Checker) Run(ctx context.Context) (*Report, error) {
	results := make([]Result, 0)

	// Run all checks
	checks := []func(context.Context) Result{
		c.checkConnection,
		c.checkKubernetesVersion,
		c.checkMilvusOperator,
		c.checkNamespace,
		c.checkStorageClass,
		c.checkResourceQuota,
	}

	for _, check := range checks {
		results = append(results, check(ctx))
	}

	// Build summary
	summary := Summary{Total: len(results)}
	canDeploy := true
	for _, r := range results {
		switch r.Status {
		case StatusPass:
			summary.Passed++
		case StatusWarn:
			summary.Warned++
		case StatusFail:
			summary.Failed++
			canDeploy = false
		}
	}

	return &Report{
		Results:   results,
		Summary:   summary,
		CanDeploy: canDeploy,
	}, nil
}

// checkConnection checks if we can connect to the Kubernetes cluster
func (c *Checker) checkConnection(ctx context.Context) Result {
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return Result{
			Name:    "Kubernetes Connection",
			Status:  StatusFail,
			Message: fmt.Sprintf("Cannot connect to Kubernetes cluster: %v", err),
			Suggest: "Check your kubeconfig file and network connectivity",
		}
	}
	return Result{
		Name:    "Kubernetes Connection",
		Status:  StatusPass,
		Message: "Successfully connected to Kubernetes cluster",
	}
}

// checkKubernetesVersion checks if the Kubernetes version is supported
func (c *Checker) checkKubernetesVersion(ctx context.Context) Result {
	serverVersion, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return Result{
			Name:    "Kubernetes Version",
			Status:  StatusFail,
			Message: fmt.Sprintf("Failed to get Kubernetes version: %v", err),
		}
	}

	// Parse version
	major, minor := parseVersion(serverVersion)

	// Milvus Operator requires Kubernetes 1.20+
	if major < 1 || (major == 1 && minor < 20) {
		return Result{
			Name:    "Kubernetes Version",
			Status:  StatusFail,
			Message: fmt.Sprintf("Kubernetes %s is not supported (requires 1.20+)", serverVersion.GitVersion),
			Suggest: "Upgrade your Kubernetes cluster to version 1.20 or later",
		}
	}

	// Warn if version is old but still supported
	if major == 1 && minor < 25 {
		return Result{
			Name:    "Kubernetes Version",
			Status:  StatusWarn,
			Message: fmt.Sprintf("Kubernetes %s is supported but consider upgrading to 1.25+", serverVersion.GitVersion),
		}
	}

	return Result{
		Name:    "Kubernetes Version",
		Status:  StatusPass,
		Message: fmt.Sprintf("Kubernetes %s is supported", serverVersion.GitVersion),
	}
}

// checkMilvusOperator checks if Milvus Operator is installed
func (c *Checker) checkMilvusOperator(ctx context.Context) Result {
	// Check if Milvus CRD exists
	_, err := c.clientset.Discovery().ServerResourcesForGroupVersion(k8s.MilvusGroup + "/" + k8s.MilvusVersion)
	if err != nil {
		return Result{
			Name:    "Milvus Operator",
			Status:  StatusFail,
			Message: "Milvus Operator is not installed (CRD not found)",
			Suggest: "Install Milvus Operator: kubectl apply -f https://raw.githubusercontent.com/zilliztech/milvus-operator/main/deploy/manifests/deployment.yaml",
		}
	}

	// Check if operator deployment exists
	deployments := []string{"milvus-operator"}
	namespaces := []string{"milvus-operator", "default", "kube-system"}

	operatorFound := false
	for _, ns := range namespaces {
		for _, name := range deployments {
			deploy, err := c.clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
			if err == nil && deploy.Status.ReadyReplicas > 0 {
				operatorFound = true
				break
			}
		}
		if operatorFound {
			break
		}
	}

	if !operatorFound {
		return Result{
			Name:    "Milvus Operator",
			Status:  StatusWarn,
			Message: "Milvus CRD found but operator deployment not detected (may be in different namespace)",
		}
	}

	return Result{
		Name:    "Milvus Operator",
		Status:  StatusPass,
		Message: "Milvus Operator is installed and running",
	}
}

// checkNamespace checks if the target namespace exists or can be created
func (c *Checker) checkNamespace(ctx context.Context) Result {
	namespace := c.opts.Namespace
	if namespace == "" {
		namespace = "milvus"
	}

	_, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		// Try to check if we have permission to create namespaces
		// This is a simple check - in real scenarios we might want more sophisticated RBAC checks
		return Result{
			Name:    "Namespace",
			Status:  StatusWarn,
			Message: fmt.Sprintf("Namespace '%s' does not exist (will be created during deployment)", namespace),
		}
	}

	return Result{
		Name:    "Namespace",
		Status:  StatusPass,
		Message: fmt.Sprintf("Namespace '%s' exists", namespace),
	}
}

// checkStorageClass checks if a suitable storage class is available
func (c *Checker) checkStorageClass(ctx context.Context) Result {
	storageClasses, err := c.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return Result{
			Name:    "Storage Class",
			Status:  StatusFail,
			Message: fmt.Sprintf("Failed to list storage classes: %v", err),
		}
	}

	if len(storageClasses.Items) == 0 {
		return Result{
			Name:    "Storage Class",
			Status:  StatusFail,
			Message: "No storage classes available",
			Suggest: "Create a storage class or use a managed Kubernetes service with default storage",
		}
	}

	// Check for specific storage class if requested
	if c.opts.StorageClass != "" {
		for _, sc := range storageClasses.Items {
			if sc.Name == c.opts.StorageClass {
				return Result{
					Name:    "Storage Class",
					Status:  StatusPass,
					Message: fmt.Sprintf("Storage class '%s' is available", c.opts.StorageClass),
				}
			}
		}
		return Result{
			Name:    "Storage Class",
			Status:  StatusFail,
			Message: fmt.Sprintf("Storage class '%s' not found", c.opts.StorageClass),
			Suggest: fmt.Sprintf("Available storage classes: %s", getStorageClassNames(storageClasses.Items)),
		}
	}

	// Check for default storage class
	for _, sc := range storageClasses.Items {
		if isDefaultStorageClass(&sc) {
			return Result{
				Name:    "Storage Class",
				Status:  StatusPass,
				Message: fmt.Sprintf("Default storage class '%s' is available", sc.Name),
			}
		}
	}

	return Result{
		Name:    "Storage Class",
		Status:  StatusWarn,
		Message: fmt.Sprintf("No default storage class found (%d storage classes available)", len(storageClasses.Items)),
		Suggest: "Specify a storage class in your topology or set a default storage class",
	}
}

// checkResourceQuota checks resource quota in the namespace
func (c *Checker) checkResourceQuota(ctx context.Context) Result {
	namespace := c.opts.Namespace
	if namespace == "" {
		namespace = "milvus"
	}

	// Check if namespace exists first
	_, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return Result{
			Name:    "Resource Quota",
			Status:  StatusPass,
			Message: "No resource quota restrictions (namespace not yet created)",
		}
	}

	quotas, err := c.clientset.CoreV1().ResourceQuotas(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return Result{
			Name:    "Resource Quota",
			Status:  StatusWarn,
			Message: fmt.Sprintf("Failed to check resource quotas: %v", err),
		}
	}

	if len(quotas.Items) == 0 {
		return Result{
			Name:    "Resource Quota",
			Status:  StatusPass,
			Message: "No resource quota restrictions in namespace",
		}
	}

	// Check if there's enough capacity
	for _, quota := range quotas.Items {
		hard := quota.Status.Hard
		used := quota.Status.Used

		// Check CPU
		if hardCPU, ok := hard["requests.cpu"]; ok {
			if usedCPU, ok := used["requests.cpu"]; ok {
				if usedCPU.Cmp(hardCPU) >= 0 {
					return Result{
						Name:    "Resource Quota",
						Status:  StatusWarn,
						Message: fmt.Sprintf("CPU quota nearly exhausted in namespace '%s'", namespace),
						Suggest: "Request more CPU quota or reduce resource requests",
					}
				}
			}
		}

		// Check Memory
		if hardMem, ok := hard["requests.memory"]; ok {
			if usedMem, ok := used["requests.memory"]; ok {
				if usedMem.Cmp(hardMem) >= 0 {
					return Result{
						Name:    "Resource Quota",
						Status:  StatusWarn,
						Message: fmt.Sprintf("Memory quota nearly exhausted in namespace '%s'", namespace),
						Suggest: "Request more memory quota or reduce resource requests",
					}
				}
			}
		}
	}

	return Result{
		Name:    "Resource Quota",
		Status:  StatusPass,
		Message: "Resource quota has sufficient capacity",
	}
}

// Helper functions

func buildConfig(kubeconfig, kubecontext string) (*rest.Config, error) {
	if kubeconfig == "" {
		// Check KUBECONFIG environment variable first
		if envKubeconfig := os.Getenv("KUBECONFIG"); envKubeconfig != "" {
			kubeconfig = envKubeconfig
		} else {
			// Try in-cluster config
			config, err := rest.InClusterConfig()
			if err == nil {
				return config, nil
			}

			// Fall back to default kubeconfig
			kubeconfig = clientcmd.RecommendedHomeFile
		}
	}

	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
	configOverrides := &clientcmd.ConfigOverrides{}
	if kubecontext != "" {
		configOverrides.CurrentContext = kubecontext
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	).ClientConfig()
}

func parseVersion(v *version.Info) (major, minor int) {
	_, _ = fmt.Sscanf(v.Major, "%d", &major)
	// Minor might have "+" suffix
	minorStr := strings.TrimSuffix(v.Minor, "+")
	_, _ = fmt.Sscanf(minorStr, "%d", &minor)
	return
}

func isDefaultStorageClass(sc *storagev1.StorageClass) bool {
	if sc.Annotations == nil {
		return false
	}
	// Check both annotations for default storage class
	if v, ok := sc.Annotations["storageclass.kubernetes.io/is-default-class"]; ok && v == "true" {
		return true
	}
	if v, ok := sc.Annotations["storageclass.beta.kubernetes.io/is-default-class"]; ok && v == "true" {
		return true
	}
	return false
}

func getStorageClassNames(classes []storagev1.StorageClass) string {
	names := make([]string, 0, len(classes))
	for _, sc := range classes {
		names = append(names, sc.Name)
	}
	return strings.Join(names, ", ")
}
