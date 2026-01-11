//go:build integration

package executor

import (
	"context"
	"os"
	"testing"
	"time"
)

// Standalone CRD template for testing
const testStandaloneCRD = `apiVersion: milvus.io/v1beta1
kind: Milvus
metadata:
  name: test-standalone
  namespace: default
spec:
  mode: standalone
  components:
    image: milvusdb/milvus:v2.5.4
  dependencies:
    etcd:
      inCluster:
        deletionPolicy: Delete
        pvcDeletion: true
        values:
          replicaCount: 1
    storage:
      inCluster:
        deletionPolicy: Delete
        pvcDeletion: true
        values:
          mode: standalone
`

func TestKubernetesCRDExecutor_Integration(t *testing.T) {
	kubeconfig := getKubeconfig()
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		t.Skip("kubeconfig not found, skipping integration test")
	}

	testName := "miup-crd-test-" + time.Now().Format("150405")
	namespace := "default"

	executor, err := NewKubernetesCRDExecutor(KubernetesCRDOptions{
		Kubeconfig:  kubeconfig,
		Namespace:   namespace,
		ClusterName: testName,
		CRDContent:  []byte(testStandaloneCRD),
	})
	if err != nil {
		t.Fatalf("Failed to create CRD executor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test parseCRD
	milvus, err := executor.parseCRD()
	if err != nil {
		t.Fatalf("parseCRD failed: %v", err)
	}

	if milvus.APIVersion != "milvus.io/v1beta1" {
		t.Errorf("Expected APIVersion 'milvus.io/v1beta1', got '%s'", milvus.APIVersion)
	}
	if milvus.Kind != "Milvus" {
		t.Errorf("Expected Kind 'Milvus', got '%s'", milvus.Kind)
	}

	// Verify we can check operator status
	_, err = executor.client.CheckMilvusOperatorInstalled(ctx)
	if err != nil {
		t.Fatalf("Failed to check Milvus Operator: %v", err)
	}

	t.Log("CRD executor basic integration test passed")
}

func TestKubernetesCRDExecutor_DeployLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	kubeconfig := getKubeconfig()
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		t.Skip("kubeconfig not found, skipping integration test")
	}

	testName := "miup-crd-lifecycle-" + time.Now().Format("150405")
	namespace := "default"

	executor, err := NewKubernetesCRDExecutor(KubernetesCRDOptions{
		Kubeconfig:  kubeconfig,
		Namespace:   namespace,
		ClusterName: testName,
		CRDContent:  []byte(testStandaloneCRD),
	})
	if err != nil {
		t.Fatalf("Failed to create CRD executor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Cleanup on exit
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cleanupCancel()
		t.Log("Cleaning up...")
		_ = executor.Destroy(cleanupCtx)
	}()

	// Deploy
	t.Log("Deploying Milvus from CRD...")
	if err := executor.Deploy(ctx); err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	t.Log("Deploy completed")

	// Check status
	t.Log("Checking status...")
	status, err := executor.Status(ctx)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	t.Logf("Status: %s", status)

	// Check if running
	t.Log("Checking if running...")
	running, err := executor.IsRunning(ctx)
	if err != nil {
		t.Fatalf("IsRunning failed: %v", err)
	}
	if !running {
		t.Errorf("Expected cluster to be running")
	}

	// Get version
	t.Log("Getting version...")
	version, err := executor.GetVersion(ctx)
	if err != nil {
		t.Logf("GetVersion warning: %v", err)
	} else {
		t.Logf("Version: %s", version)
		if version != "v2.5.4" {
			t.Errorf("Expected version 'v2.5.4', got '%s'", version)
		}
	}

	// Get replicas
	t.Log("Getting replicas...")
	replicas, err := executor.GetReplicas(ctx)
	if err != nil {
		t.Fatalf("GetReplicas failed: %v", err)
	}
	t.Logf("Replicas: %+v", replicas)
	if replicas["standalone"] != 1 {
		t.Errorf("Expected standalone replicas 1, got %d", replicas["standalone"])
	}

	// Get config
	t.Log("Getting config...")
	config, err := executor.GetConfig(ctx)
	if err != nil {
		t.Logf("GetConfig warning: %v", err)
	} else {
		t.Logf("Config: %+v", config)
	}

	// Diagnose
	t.Log("Running diagnose...")
	result, err := executor.Diagnose(ctx)
	if err != nil {
		t.Fatalf("Diagnose failed: %v", err)
	}
	t.Logf("Diagnose result: healthy=%v, summary=%s", result.Healthy, result.Summary)
	if !result.Healthy {
		t.Errorf("Expected cluster to be healthy")
	}

	// Get logs
	t.Log("Getting logs...")
	logs, err := executor.Logs(ctx, "", 10)
	if err != nil {
		t.Logf("Logs warning: %v", err)
	} else {
		t.Logf("Logs length: %d bytes", len(logs))
	}

	// Stop
	t.Log("Stopping...")
	if err := executor.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Wait a bit for stop to take effect
	time.Sleep(5 * time.Second)

	// Start
	t.Log("Starting...")
	if err := executor.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify running again
	running, err = executor.IsRunning(ctx)
	if err != nil {
		t.Fatalf("IsRunning after start failed: %v", err)
	}
	if !running {
		t.Errorf("Expected cluster to be running after start")
	}

	// Destroy
	t.Log("Destroying...")
	if err := executor.Destroy(ctx); err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	t.Log("CRD executor lifecycle integration test completed successfully")
}

func TestKubernetesCRDExecutor_ParseCRD(t *testing.T) {
	testCases := []struct {
		name        string
		crd         string
		expectError bool
		checkName   string
	}{
		{
			name: "valid standalone CRD",
			crd: `apiVersion: milvus.io/v1beta1
kind: Milvus
metadata:
  name: test-milvus
  namespace: default
spec:
  mode: standalone`,
			expectError: false,
			checkName:   "test-milvus",
		},
		{
			name: "valid CRD with labels",
			crd: `apiVersion: milvus.io/v1beta1
kind: Milvus
metadata:
  name: labeled-milvus
  namespace: milvus
  labels:
    app: milvus
    env: test
spec:
  mode: standalone`,
			expectError: false,
			checkName:   "labeled-milvus",
		},
		{
			name: "invalid kind",
			crd: `apiVersion: milvus.io/v1beta1
kind: NotMilvus
metadata:
  name: test-milvus
spec:
  mode: standalone`,
			expectError: true,
		},
		{
			name:        "invalid YAML",
			crd:         `not: valid: yaml: content`,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			executor := &KubernetesCRDExecutor{
				crdContent: []byte(tc.crd),
			}

			milvus, err := executor.parseCRD()
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if milvus.Name != tc.checkName {
				t.Errorf("Expected name '%s', got '%s'", tc.checkName, milvus.Name)
			}

			if milvus.APIVersion != "milvus.io/v1beta1" {
				t.Errorf("Expected APIVersion 'milvus.io/v1beta1', got '%s'", milvus.APIVersion)
			}

			if milvus.Kind != "Milvus" {
				t.Errorf("Expected Kind 'Milvus', got '%s'", milvus.Kind)
			}
		})
	}
}
