package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps Kubernetes client operations
type Client struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
	config        *rest.Config
	namespace     string
}

// ClientOptions contains options for creating a client
type ClientOptions struct {
	Kubeconfig string
	Context    string
	Namespace  string
}

// NewClient creates a new Kubernetes client
func NewClient(opts ClientOptions) (*Client, error) {
	config, err := buildConfig(opts.Kubeconfig, opts.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	namespace := opts.Namespace
	if namespace == "" {
		namespace = "default"
	}

	return &Client{
		clientset:     clientset,
		dynamicClient: dynamicClient,
		config:        config,
		namespace:     namespace,
	}, nil
}

// buildConfig builds a Kubernetes config from kubeconfig file
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
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			kubeconfig = filepath.Join(home, ".kube", "config")
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

// milvusGVR returns the GroupVersionResource for Milvus
func milvusGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    MilvusGroup,
		Version:  MilvusVersion,
		Resource: MilvusResource,
	}
}

// CreateMilvus creates a Milvus resource
func (c *Client) CreateMilvus(ctx context.Context, milvus *Milvus) error {
	obj, err := toUnstructured(milvus)
	if err != nil {
		return fmt.Errorf("failed to convert to unstructured: %w", err)
	}

	namespace := milvus.Namespace
	if namespace == "" {
		namespace = c.namespace
	}

	_, err = c.dynamicClient.Resource(milvusGVR()).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Milvus: %w", err)
	}

	return nil
}

// GetMilvus gets a Milvus resource
func (c *Client) GetMilvus(ctx context.Context, name, namespace string) (*Milvus, error) {
	if namespace == "" {
		namespace = c.namespace
	}

	obj, err := c.dynamicClient.Resource(milvusGVR()).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Milvus: %w", err)
	}

	milvus, err := fromUnstructured(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to convert from unstructured: %w", err)
	}

	return milvus, nil
}

// UpdateMilvus updates a Milvus resource
func (c *Client) UpdateMilvus(ctx context.Context, milvus *Milvus) error {
	obj, err := toUnstructured(milvus)
	if err != nil {
		return fmt.Errorf("failed to convert to unstructured: %w", err)
	}

	namespace := milvus.Namespace
	if namespace == "" {
		namespace = c.namespace
	}

	_, err = c.dynamicClient.Resource(milvusGVR()).Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update Milvus: %w", err)
	}

	return nil
}

// DeleteMilvus deletes a Milvus resource
func (c *Client) DeleteMilvus(ctx context.Context, name, namespace string) error {
	if namespace == "" {
		namespace = c.namespace
	}

	err := c.dynamicClient.Resource(milvusGVR()).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete Milvus: %w", err)
	}

	return nil
}

// ListMilvus lists all Milvus resources in a namespace
func (c *Client) ListMilvus(ctx context.Context, namespace string) (*MilvusList, error) {
	if namespace == "" {
		namespace = c.namespace
	}

	list, err := c.dynamicClient.Resource(milvusGVR()).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list Milvus: %w", err)
	}

	milvusList := &MilvusList{
		Items: make([]Milvus, 0, len(list.Items)),
	}

	for _, item := range list.Items {
		milvus, err := fromUnstructured(&item)
		if err != nil {
			continue
		}
		milvusList.Items = append(milvusList.Items, *milvus)
	}

	return milvusList, nil
}

// ListAllMilvus lists all Milvus resources across all namespaces
func (c *Client) ListAllMilvus(ctx context.Context) (*MilvusList, error) {
	list, err := c.dynamicClient.Resource(milvusGVR()).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list all Milvus: %w", err)
	}

	milvusList := &MilvusList{
		Items: make([]Milvus, 0, len(list.Items)),
	}

	for _, item := range list.Items {
		milvus, err := fromUnstructured(&item)
		if err != nil {
			continue
		}
		milvusList.Items = append(milvusList.Items, *milvus)
	}

	return milvusList, nil
}

// GetPodLogs gets logs from a pod
func (c *Client) GetPodLogs(ctx context.Context, namespace, podName, container string, tailLines int64) (string, error) {
	if namespace == "" {
		namespace = c.namespace
	}

	opts := &corev1.PodLogOptions{
		TailLines: &tailLines,
	}
	if container != "" {
		opts.Container = container
	}

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
	logs, err := req.DoRaw(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get pod logs: %w", err)
	}

	return string(logs), nil
}

// GetMilvusPods gets pods for a Milvus cluster
func (c *Client) GetMilvusPods(ctx context.Context, name, namespace string) ([]string, error) {
	if namespace == "" {
		namespace = c.namespace
	}

	labelSelector := fmt.Sprintf("app.kubernetes.io/instance=%s", name)
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	result := make([]string, 0, len(pods.Items))
	for _, pod := range pods.Items {
		result = append(result, pod.Name)
	}

	return result, nil
}

// GetMilvusService gets the service endpoint for a Milvus cluster
func (c *Client) GetMilvusService(ctx context.Context, name, namespace string) (string, error) {
	if namespace == "" {
		namespace = c.namespace
	}

	svc, err := c.clientset.CoreV1().Services(namespace).Get(ctx, name+"-milvus", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get service: %w", err)
	}

	if len(svc.Spec.Ports) == 0 {
		return "", fmt.Errorf("no ports found in service")
	}

	// Return cluster IP and port
	port := svc.Spec.Ports[0].Port
	return fmt.Sprintf("%s:%d", svc.Spec.ClusterIP, port), nil
}

// CheckMilvusOperatorInstalled checks if Milvus Operator is installed
func (c *Client) CheckMilvusOperatorInstalled(ctx context.Context) (bool, error) {
	// Check if Milvus CRD exists
	_, err := c.clientset.Discovery().ServerResourcesForGroupVersion(MilvusGroup + "/" + MilvusVersion)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// Namespace returns the default namespace
func (c *Client) Namespace() string {
	return c.namespace
}

// toUnstructured converts a Milvus object to unstructured
func toUnstructured(milvus *Milvus) (*unstructured.Unstructured, error) {
	data, err := json.Marshal(milvus)
	if err != nil {
		return nil, err
	}

	obj := &unstructured.Unstructured{}
	if err := json.Unmarshal(data, &obj.Object); err != nil {
		return nil, err
	}

	obj.SetAPIVersion(MilvusGroup + "/" + MilvusVersion)
	obj.SetKind(MilvusKind)

	return obj, nil
}

// fromUnstructured converts an unstructured object to Milvus
func fromUnstructured(obj *unstructured.Unstructured) (*Milvus, error) {
	data, err := json.Marshal(obj.Object)
	if err != nil {
		return nil, err
	}

	milvus := &Milvus{}
	if err := json.Unmarshal(data, milvus); err != nil {
		return nil, err
	}

	return milvus, nil
}
