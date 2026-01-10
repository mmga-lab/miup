package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mmga-lab/miup/pkg/localdata"
)

const (
	// AuditDirName is the directory name for audit logs
	AuditDirName = "audit"
	// AuditFileName is the audit log file name
	AuditFileName = "audit.log"
)

// Status represents the status of an operation
type Status string

const (
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
	StatusRunning Status = "running"
)

// Entry represents a single audit log entry
type Entry struct {
	ID           string        `json:"id"`
	Timestamp    time.Time     `json:"timestamp"`
	Instance     string        `json:"instance,omitempty"`
	Command      string        `json:"command"`
	Args         []string      `json:"args,omitempty"`
	User         string        `json:"user,omitempty"`
	Status       Status        `json:"status"`
	Duration     time.Duration `json:"duration,omitempty"`
	Error        string        `json:"error,omitempty"`
	Message      string        `json:"message,omitempty"`
}

// Logger handles audit logging
type Logger struct {
	mu       sync.Mutex
	filePath string
}

// NewLogger creates a new audit logger
func NewLogger() (*Logger, error) {
	profile, err := localdata.DefaultProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	auditDir := profile.Path(AuditDirName)
	if err := os.MkdirAll(auditDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit directory: %w", err)
	}

	return &Logger{
		filePath: filepath.Join(auditDir, AuditFileName),
	}, nil
}

// NewLoggerWithPath creates an audit logger with a custom path
func NewLoggerWithPath(path string) *Logger {
	return &Logger{filePath: path}
}

// Log writes an audit entry to the log file
func (l *Logger) Log(entry *Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Set timestamp if not set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Generate ID if not set
	if entry.ID == "" {
		entry.ID = generateID(entry.Timestamp)
	}

	// Get current user
	if entry.User == "" {
		entry.User = getCurrentUser()
	}

	// Open file in append mode
	f, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit log: %w", err)
	}
	defer f.Close()

	// Write JSON line
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write audit entry: %w", err)
	}

	return nil
}

// LogOperation is a convenience method to log an operation with timing
func (l *Logger) LogOperation(instance, command string, args []string, fn func() error) error {
	entry := &Entry{
		Timestamp: time.Now(),
		Instance:  instance,
		Command:   command,
		Args:      args,
		Status:    StatusRunning,
	}

	start := time.Now()
	err := fn()
	entry.Duration = time.Since(start)

	if err != nil {
		entry.Status = StatusFailed
		entry.Error = err.Error()
	} else {
		entry.Status = StatusSuccess
	}

	if logErr := l.Log(entry); logErr != nil {
		// Log error but don't fail the operation
		fmt.Fprintf(os.Stderr, "Warning: failed to write audit log: %v\n", logErr)
	}

	return err
}

// QueryOptions contains options for querying audit logs
type QueryOptions struct {
	Instance  string     // Filter by instance name
	Command   string     // Filter by command
	Status    Status     // Filter by status
	StartTime *time.Time // Filter by start time
	EndTime   *time.Time // Filter by end time
	Limit     int        // Maximum number of entries to return (0 = unlimited)
}

// Query reads and filters audit log entries
func (l *Logger) Query(opts QueryOptions) ([]Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.Open(l.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	// Increase buffer size for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			// Skip malformed entries
			continue
		}

		// Apply filters
		if !matchesFilter(entry, opts) {
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read audit log: %w", err)
	}

	// Apply limit (return last N entries)
	if opts.Limit > 0 && len(entries) > opts.Limit {
		entries = entries[len(entries)-opts.Limit:]
	}

	return entries, nil
}

// GetLatest returns the latest N audit entries
func (l *Logger) GetLatest(n int) ([]Entry, error) {
	return l.Query(QueryOptions{Limit: n})
}

// GetByInstance returns audit entries for a specific instance
func (l *Logger) GetByInstance(instance string, limit int) ([]Entry, error) {
	return l.Query(QueryOptions{Instance: instance, Limit: limit})
}

// Clear clears all audit logs
func (l *Logger) Clear() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return os.Remove(l.filePath)
}

// matchesFilter checks if an entry matches the query options
func matchesFilter(entry Entry, opts QueryOptions) bool {
	if opts.Instance != "" && entry.Instance != opts.Instance {
		return false
	}
	if opts.Command != "" && entry.Command != opts.Command {
		return false
	}
	if opts.Status != "" && entry.Status != opts.Status {
		return false
	}
	if opts.StartTime != nil && entry.Timestamp.Before(*opts.StartTime) {
		return false
	}
	if opts.EndTime != nil && entry.Timestamp.After(*opts.EndTime) {
		return false
	}
	return true
}

// generateID generates a unique ID for an audit entry
func generateID(t time.Time) string {
	return fmt.Sprintf("%d", t.UnixNano())
}

// getCurrentUser returns the current username
func getCurrentUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "unknown"
}

// FilePath returns the audit log file path
func (l *Logger) FilePath() string {
	return l.filePath
}
