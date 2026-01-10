package audit

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogger_Log(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewLoggerWithPath(logPath)

	entry := &Entry{
		Instance: "test-instance",
		Command:  "deploy",
		Args:     []string{"--kubeconfig", "/path/to/config"},
		Status:   StatusSuccess,
		Message:  "Deployment successful",
	}

	err = logger.Log(entry)
	if err != nil {
		t.Fatalf("failed to log entry: %v", err)
	}

	// Verify file was created and contains data
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if len(data) == 0 {
		t.Error("log file is empty")
	}
}

func TestLogger_Query(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewLoggerWithPath(logPath)

	// Log multiple entries
	entries := []Entry{
		{Instance: "instance-1", Command: "deploy", Status: StatusSuccess},
		{Instance: "instance-1", Command: "start", Status: StatusSuccess},
		{Instance: "instance-2", Command: "deploy", Status: StatusFailed, Error: "connection failed"},
		{Instance: "instance-1", Command: "stop", Status: StatusSuccess},
	}

	for i := range entries {
		if err := logger.Log(&entries[i]); err != nil {
			t.Fatalf("failed to log entry: %v", err)
		}
	}

	// Query all entries
	result, err := logger.Query(QueryOptions{})
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if len(result) != 4 {
		t.Errorf("expected 4 entries, got %d", len(result))
	}

	// Query by instance
	result, err = logger.Query(QueryOptions{Instance: "instance-1"})
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 entries for instance-1, got %d", len(result))
	}

	// Query by command
	result, err = logger.Query(QueryOptions{Command: "deploy"})
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 deploy entries, got %d", len(result))
	}

	// Query by status
	result, err = logger.Query(QueryOptions{Status: StatusFailed})
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 failed entry, got %d", len(result))
	}
}

func TestLogger_QueryWithLimit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewLoggerWithPath(logPath)

	// Log 10 entries
	for i := range 10 {
		entry := &Entry{
			Instance: "test",
			Command:  "test",
			Status:   StatusSuccess,
			Message:  string(rune('A' + i)),
		}
		if err := logger.Log(entry); err != nil {
			t.Fatalf("failed to log entry: %v", err)
		}
	}

	// Query with limit
	result, err := logger.Query(QueryOptions{Limit: 3})
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 entries, got %d", len(result))
	}

	// Should return last 3 entries (H, I, J)
	if result[0].Message != "H" || result[1].Message != "I" || result[2].Message != "J" {
		t.Errorf("expected last 3 entries, got %v, %v, %v", result[0].Message, result[1].Message, result[2].Message)
	}
}

func TestLogger_QueryWithTimeRange(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewLoggerWithPath(logPath)

	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	// Log entries with different timestamps
	entries := []Entry{
		{Timestamp: past, Instance: "test", Command: "old", Status: StatusSuccess},
		{Timestamp: now, Instance: "test", Command: "current", Status: StatusSuccess},
	}

	for i := range entries {
		if err := logger.Log(&entries[i]); err != nil {
			t.Fatalf("failed to log entry: %v", err)
		}
	}

	// Query with start time (should exclude old entry)
	halfHourAgo := now.Add(-30 * time.Minute)
	result, err := logger.Query(QueryOptions{StartTime: &halfHourAgo})
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 entry, got %d", len(result))
	}

	// Query with end time (should include both)
	result, err = logger.Query(QueryOptions{EndTime: &future})
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
}

func TestLogger_GetLatest(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewLoggerWithPath(logPath)

	// Log 5 entries
	for range 5 {
		entry := &Entry{Instance: "test", Command: "test", Status: StatusSuccess}
		if err := logger.Log(entry); err != nil {
			t.Fatalf("failed to log entry: %v", err)
		}
	}

	result, err := logger.GetLatest(2)
	if err != nil {
		t.Fatalf("failed to get latest: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
}

func TestLogger_GetByInstance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewLoggerWithPath(logPath)

	// Log entries for different instances
	_ = logger.Log(&Entry{Instance: "prod", Command: "deploy", Status: StatusSuccess})
	_ = logger.Log(&Entry{Instance: "dev", Command: "deploy", Status: StatusSuccess})
	_ = logger.Log(&Entry{Instance: "prod", Command: "scale", Status: StatusSuccess})

	result, err := logger.GetByInstance("prod", 10)
	if err != nil {
		t.Fatalf("failed to get by instance: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries for prod, got %d", len(result))
	}
}

func TestLogger_LogOperation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewLoggerWithPath(logPath)

	// Test successful operation
	err = logger.LogOperation("test-instance", "deploy", []string{"--debug"}, func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatalf("LogOperation failed: %v", err)
	}

	// Test failed operation
	expectedErr := errors.New("deployment failed")
	err = logger.LogOperation("test-instance", "deploy", nil, func() error {
		return expectedErr
	})
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	// Verify entries
	entries, err := logger.Query(QueryOptions{})
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// First entry should be success
	if entries[0].Status != StatusSuccess {
		t.Errorf("expected success status, got %s", entries[0].Status)
	}
	if entries[0].Duration < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", entries[0].Duration)
	}

	// Second entry should be failed
	if entries[1].Status != StatusFailed {
		t.Errorf("expected failed status, got %s", entries[1].Status)
	}
	if entries[1].Error != expectedErr.Error() {
		t.Errorf("expected error %q, got %q", expectedErr.Error(), entries[1].Error)
	}
}

func TestLogger_Clear(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewLoggerWithPath(logPath)

	// Log an entry
	_ = logger.Log(&Entry{Instance: "test", Command: "test", Status: StatusSuccess})

	// Clear
	err = logger.Clear()
	if err != nil {
		t.Fatalf("failed to clear: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}

	// Query should return empty
	result, err := logger.Query(QueryOptions{})
	if err != nil {
		t.Fatalf("failed to query after clear: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(result))
	}
}

func TestLogger_QueryEmptyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "audit.log")
	logger := NewLoggerWithPath(logPath)

	// Query non-existent file should return empty slice
	result, err := logger.Query(QueryOptions{})
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result))
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusSuccess != "success" {
		t.Errorf("StatusSuccess = %q, want %q", StatusSuccess, "success")
	}
	if StatusFailed != "failed" {
		t.Errorf("StatusFailed = %q, want %q", StatusFailed, "failed")
	}
	if StatusRunning != "running" {
		t.Errorf("StatusRunning = %q, want %q", StatusRunning, "running")
	}
}

func TestGenerateID(t *testing.T) {
	t1 := time.Now()
	t2 := t1.Add(1 * time.Nanosecond)

	id1 := generateID(t1)
	id2 := generateID(t2)

	if id1 == id2 {
		t.Error("IDs should be different for different times")
	}
}

func TestGetCurrentUser(t *testing.T) {
	user := getCurrentUser()
	if user == "" {
		t.Error("getCurrentUser returned empty string")
	}
}

func TestLogger_FilePath(t *testing.T) {
	logger := NewLoggerWithPath("/test/path/audit.log")
	if logger.FilePath() != "/test/path/audit.log" {
		t.Errorf("FilePath = %q, want %q", logger.FilePath(), "/test/path/audit.log")
	}
}
