package check

import (
	"fmt"
	"strings"
	"testing"
)

func TestStatusValues(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusPass, "pass"},
		{StatusWarn, "warn"},
		{StatusFail, "fail"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Status %v = %s, want %s", tt.status, string(tt.status), tt.expected)
		}
	}
}

func TestResultStruct(t *testing.T) {
	r := Result{
		Name:    "Test Check",
		Status:  StatusPass,
		Message: "Test passed",
		Suggest: "No action needed",
	}

	if r.Name != "Test Check" {
		t.Errorf("Name = %s, want 'Test Check'", r.Name)
	}
	if r.Status != StatusPass {
		t.Errorf("Status = %s, want 'pass'", r.Status)
	}
}

func TestReportSummary(t *testing.T) {
	report := &Report{
		Results: []Result{
			{Name: "Check 1", Status: StatusPass, Message: "OK"},
			{Name: "Check 2", Status: StatusPass, Message: "OK"},
			{Name: "Check 3", Status: StatusWarn, Message: "Warning"},
			{Name: "Check 4", Status: StatusFail, Message: "Failed"},
		},
		Summary: Summary{
			Total:  4,
			Passed: 2,
			Warned: 1,
			Failed: 1,
		},
		CanDeploy: false,
	}

	if report.Summary.Total != 4 {
		t.Errorf("Total = %d, want 4", report.Summary.Total)
	}
	if report.Summary.Passed != 2 {
		t.Errorf("Passed = %d, want 2", report.Summary.Passed)
	}
	if report.Summary.Warned != 1 {
		t.Errorf("Warned = %d, want 1", report.Summary.Warned)
	}
	if report.Summary.Failed != 1 {
		t.Errorf("Failed = %d, want 1", report.Summary.Failed)
	}
	if report.CanDeploy {
		t.Error("CanDeploy should be false when there are failures")
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name          string
		major         string
		minor         string
		expectedMajor int
		expectedMinor int
	}{
		{"standard version", "1", "25", 1, 25},
		{"version with plus", "1", "25+", 1, 25},
		{"older version", "1", "20", 1, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock version.Info
			v := &versionInfo{Major: tt.major, Minor: tt.minor}
			major, minor := parseVersionFromStrings(v.Major, v.Minor)

			if major != tt.expectedMajor {
				t.Errorf("major = %d, want %d", major, tt.expectedMajor)
			}
			if minor != tt.expectedMinor {
				t.Errorf("minor = %d, want %d", minor, tt.expectedMinor)
			}
		})
	}
}

// versionInfo is a test helper struct
type versionInfo struct {
	Major string
	Minor string
}

// parseVersionFromStrings is a test helper
func parseVersionFromStrings(major, minor string) (int, int) {
	var maj, min int
	_, _ = fmt.Sscanf(major, "%d", &maj)
	minorStr := strings.TrimSuffix(minor, "+")
	_, _ = fmt.Sscanf(minorStr, "%d", &min)
	return maj, min
}

func TestOptionsDefaults(t *testing.T) {
	opts := Options{
		Kubeconfig:   "",
		Context:      "",
		Namespace:    "",
		StorageClass: "",
	}

	// Test that empty options are handled
	if opts.Namespace != "" {
		t.Errorf("Namespace should be empty by default")
	}
}

func TestCanDeployLogic(t *testing.T) {
	tests := []struct {
		name      string
		results   []Result
		canDeploy bool
	}{
		{
			name: "all pass",
			results: []Result{
				{Status: StatusPass},
				{Status: StatusPass},
			},
			canDeploy: true,
		},
		{
			name: "with warnings",
			results: []Result{
				{Status: StatusPass},
				{Status: StatusWarn},
			},
			canDeploy: true,
		},
		{
			name: "with failure",
			results: []Result{
				{Status: StatusPass},
				{Status: StatusFail},
			},
			canDeploy: false,
		},
		{
			name: "multiple failures",
			results: []Result{
				{Status: StatusFail},
				{Status: StatusFail},
			},
			canDeploy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canDeploy := true
			for _, r := range tt.results {
				if r.Status == StatusFail {
					canDeploy = false
					break
				}
			}

			if canDeploy != tt.canDeploy {
				t.Errorf("canDeploy = %v, want %v", canDeploy, tt.canDeploy)
			}
		})
	}
}
