package executor

import (
	"slices"
	"testing"
	"time"
)

func TestCheckStatusConstants(t *testing.T) {
	if CheckStatusOK != "OK" {
		t.Errorf("CheckStatusOK = %s, want OK", CheckStatusOK)
	}
	if CheckStatusWarning != "WARNING" {
		t.Errorf("CheckStatusWarning = %s, want WARNING", CheckStatusWarning)
	}
	if CheckStatusError != "ERROR" {
		t.Errorf("CheckStatusError = %s, want ERROR", CheckStatusError)
	}
}

func TestScaleOptions_HasReplicaChange(t *testing.T) {
	tests := []struct {
		name     string
		opts     ScaleOptions
		expected bool
	}{
		{"zero replicas", ScaleOptions{Replicas: 0}, false},
		{"positive replicas", ScaleOptions{Replicas: 3}, true},
		{"one replica", ScaleOptions{Replicas: 1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.HasReplicaChange(); got != tt.expected {
				t.Errorf("HasReplicaChange() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestScaleOptions_HasResourceChange(t *testing.T) {
	tests := []struct {
		name     string
		opts     ScaleOptions
		expected bool
	}{
		{"empty", ScaleOptions{}, false},
		{"only replicas", ScaleOptions{Replicas: 3}, false},
		{"cpu request", ScaleOptions{CPURequest: "2"}, true},
		{"cpu limit", ScaleOptions{CPULimit: "4"}, true},
		{"memory request", ScaleOptions{MemoryRequest: "4Gi"}, true},
		{"memory limit", ScaleOptions{MemoryLimit: "8Gi"}, true},
		{"all resources", ScaleOptions{
			CPURequest:    "2",
			CPULimit:      "4",
			MemoryRequest: "4Gi",
			MemoryLimit:   "8Gi",
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.HasResourceChange(); got != tt.expected {
				t.Errorf("HasResourceChange() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestComponentNames(t *testing.T) {
	expectedComponents := []string{
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

	if len(ComponentNames) != len(expectedComponents) {
		t.Errorf("ComponentNames length = %d, want %d", len(ComponentNames), len(expectedComponents))
	}

	for _, expected := range expectedComponents {
		if !slices.Contains(ComponentNames, expected) {
			t.Errorf("ComponentNames missing %s", expected)
		}
	}
}

func TestDiagnoseResult(t *testing.T) {
	result := DiagnoseResult{
		Healthy: true,
		Summary: "All components healthy",
		Components: []ComponentCheck{
			{Name: "standalone", Status: CheckStatusOK, Message: "Running", Replicas: 1, Ready: 1},
		},
		Connectivity: []ConnectivityCheck{
			{Name: "etcd", Target: "etcd:2379", Status: CheckStatusOK, Latency: "5ms", Message: "Connected"},
		},
		Resources: []ResourceCheck{
			{Name: "memory", Status: CheckStatusOK, Usage: "2Gi", Limit: "8Gi", Message: "25% used"},
		},
		Issues: []Issue{},
	}

	if !result.Healthy {
		t.Error("Healthy should be true")
	}
	if result.Summary != "All components healthy" {
		t.Errorf("Summary = %s, want 'All components healthy'", result.Summary)
	}
	if len(result.Components) != 1 {
		t.Errorf("Components length = %d, want 1", len(result.Components))
	}
	if len(result.Connectivity) != 1 {
		t.Errorf("Connectivity length = %d, want 1", len(result.Connectivity))
	}
	if len(result.Resources) != 1 {
		t.Errorf("Resources length = %d, want 1", len(result.Resources))
	}
	if len(result.Issues) != 0 {
		t.Errorf("Issues length = %d, want 0", len(result.Issues))
	}
}

func TestComponentCheck(t *testing.T) {
	check := ComponentCheck{
		Name:     "querynode",
		Status:   CheckStatusOK,
		Message:  "3/3 replicas ready",
		Replicas: 3,
		Ready:    3,
	}

	if check.Name != "querynode" {
		t.Errorf("Name = %s, want querynode", check.Name)
	}
	if check.Status != CheckStatusOK {
		t.Errorf("Status = %s, want OK", check.Status)
	}
	if check.Message != "3/3 replicas ready" {
		t.Errorf("Message = %s, want '3/3 replicas ready'", check.Message)
	}
	if check.Replicas != 3 {
		t.Errorf("Replicas = %d, want 3", check.Replicas)
	}
	if check.Ready != 3 {
		t.Errorf("Ready = %d, want 3", check.Ready)
	}
}

func TestConnectivityCheck(t *testing.T) {
	check := ConnectivityCheck{
		Name:    "minio",
		Target:  "minio:9000",
		Status:  CheckStatusOK,
		Latency: "10ms",
		Message: "Storage accessible",
	}

	if check.Name != "minio" {
		t.Errorf("Name = %s, want minio", check.Name)
	}
	if check.Target != "minio:9000" {
		t.Errorf("Target = %s, want minio:9000", check.Target)
	}
	if check.Status != CheckStatusOK {
		t.Errorf("Status = %s, want OK", check.Status)
	}
	if check.Latency != "10ms" {
		t.Errorf("Latency = %s, want 10ms", check.Latency)
	}
	if check.Message != "Storage accessible" {
		t.Errorf("Message = %s, want 'Storage accessible'", check.Message)
	}
}

func TestResourceCheck(t *testing.T) {
	check := ResourceCheck{
		Name:    "cpu",
		Status:  CheckStatusWarning,
		Usage:   "3.5",
		Limit:   "4",
		Message: "CPU usage at 87.5%",
	}

	if check.Name != "cpu" {
		t.Errorf("Name = %s, want cpu", check.Name)
	}
	if check.Status != CheckStatusWarning {
		t.Errorf("Status = %s, want WARNING", check.Status)
	}
	if check.Usage != "3.5" {
		t.Errorf("Usage = %s, want 3.5", check.Usage)
	}
	if check.Limit != "4" {
		t.Errorf("Limit = %s, want 4", check.Limit)
	}
	if check.Message != "CPU usage at 87.5%" {
		t.Errorf("Message = %s, want 'CPU usage at 87.5%%'", check.Message)
	}
}

func TestIssue(t *testing.T) {
	issue := Issue{
		Severity:    CheckStatusError,
		Component:   "etcd",
		Description: "Etcd cluster unhealthy",
		Suggestion:  "Check etcd pod logs",
	}

	if issue.Severity != CheckStatusError {
		t.Errorf("Severity = %s, want ERROR", issue.Severity)
	}
	if issue.Component != "etcd" {
		t.Errorf("Component = %s, want etcd", issue.Component)
	}
	if issue.Description != "Etcd cluster unhealthy" {
		t.Errorf("Description = %s, want 'Etcd cluster unhealthy'", issue.Description)
	}
	if issue.Suggestion != "Check etcd pod logs" {
		t.Errorf("Suggestion = %s, want 'Check etcd pod logs'", issue.Suggestion)
	}
}

func TestReloadOptions(t *testing.T) {
	opts := ReloadOptions{
		Config: map[string]any{
			"log.level": "debug",
		},
		Wait:    true,
		Timeout: 5 * time.Minute,
	}

	if opts.Config == nil {
		t.Error("Config should not be nil")
	}
	if opts.Config["log.level"] != "debug" {
		t.Errorf("Config[log.level] = %v, want debug", opts.Config["log.level"])
	}
	if !opts.Wait {
		t.Error("Wait should be true")
	}
	if opts.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want 5m", opts.Timeout)
	}
}
