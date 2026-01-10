//go:build integration

package executor

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/mmga-lab/miup/pkg/cluster/spec"
	"github.com/mmga-lab/miup/pkg/k8s"
)

func getKubeconfig() string {
	if kc := os.Getenv("KUBECONFIG"); kc != "" {
		return kc
	}
	home, _ := os.UserHomeDir()
	return home + "/.kube/config"
}

func TestKubernetesExecutor_Integration(t *testing.T) {
	kubeconfig := getKubeconfig()
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		t.Skip("kubeconfig not found, skipping integration test")
	}

	client, err := k8s.NewClient(kubeconfig)
	if err != nil {
		t.Fatalf("Failed to create k8s client: %v", err)
	}

	// Test cluster connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Verify we can connect to the cluster
	_, err = client.GetMilvusList(ctx, "default")
	if err != nil {
		t.Fatalf("Failed to list Milvus resources: %v", err)
	}
}

func TestKubernetesExecutor_DeployStandalone(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	kubeconfig := getKubeconfig()
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		t.Skip("kubeconfig not found, skipping integration test")
	}

	client, err := k8s.NewClient(kubeconfig)
	if err != nil {
		t.Fatalf("Failed to create k8s client: %v", err)
	}

	testName := "miup-test-" + time.Now().Format("150405")
	namespace := "default"

	topology := &spec.Specification{
		GlobalOptions: spec.GlobalOptions{
			Mode:      "standalone",
			Namespace: namespace,
		},
		MilvusServers: []*spec.MilvusSpec{
			{
				Host:    testName,
				Version: "v2.5.4",
			},
		},
	}

	executor := NewKubernetesExecutor(client, testName, namespace, topology)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Cleanup on exit
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cleanupCancel()
		_ = executor.Destroy(cleanupCtx)
	}()

	// Deploy
	t.Log("Deploying standalone Milvus...")
	if err := executor.Deploy(ctx); err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Check status
	t.Log("Checking status...")
	status, err := executor.Status(ctx)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	t.Logf("Status: %s", status)

	// Get version
	t.Log("Getting version...")
	version, err := executor.GetVersion(ctx)
	if err != nil {
		t.Logf("GetVersion warning: %v", err)
	} else {
		t.Logf("Version: %s", version)
	}

	// Get replicas
	t.Log("Getting replicas...")
	replicas, err := executor.GetReplicas(ctx)
	if err != nil {
		t.Fatalf("GetReplicas failed: %v", err)
	}
	t.Logf("Replicas: %+v", replicas)

	// Stop
	t.Log("Stopping...")
	if err := executor.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Start
	t.Log("Starting...")
	if err := executor.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Destroy
	t.Log("Destroying...")
	if err := executor.Destroy(ctx); err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	t.Log("Integration test completed successfully")
}

func TestKubernetesExecutor_MilvusOperatorExists(t *testing.T) {
	kubeconfig := getKubeconfig()
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		t.Skip("kubeconfig not found, skipping integration test")
	}

	client, err := k8s.NewClient(kubeconfig)
	if err != nil {
		t.Fatalf("Failed to create k8s client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if Milvus CRD is installed
	_, err = client.GetMilvusList(ctx, "default")
	if err != nil {
		t.Logf("Milvus Operator may not be installed: %v", err)
		t.Skip("Milvus Operator not installed")
	}

	t.Log("Milvus Operator is installed")
}
