package output

import (
	"errors"
	"testing"
)

func TestStructuredError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *StructuredError
		expected string
	}{
		{
			name:     "without details",
			err:      NewError(ErrNotFound, "resource not found"),
			expected: "resource not found",
		},
		{
			name:     "with details",
			err:      NewErrorWithDetails(ErrNotFound, "resource not found", "id=123"),
			expected: "resource not found: id=123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewError(t *testing.T) {
	err := NewError(ErrTimeout, "operation timed out")

	if err.Code != ErrTimeout {
		t.Errorf("expected code %s, got %s", ErrTimeout, err.Code)
	}
	if err.Message != "operation timed out" {
		t.Errorf("expected message 'operation timed out', got '%s'", err.Message)
	}
	if err.Details != "" {
		t.Errorf("expected empty details, got '%s'", err.Details)
	}
}

func TestNewErrorWithDetails(t *testing.T) {
	err := NewErrorWithDetails(ErrPermission, "access denied", "user=admin")

	if err.Code != ErrPermission {
		t.Errorf("expected code %s, got %s", ErrPermission, err.Code)
	}
	if err.Details != "user=admin" {
		t.Errorf("expected details 'user=admin', got '%s'", err.Details)
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("connection refused")
	wrapped := WrapError(ErrK8sConnection, originalErr)

	if wrapped.Code != ErrK8sConnection {
		t.Errorf("expected code %s, got %s", ErrK8sConnection, wrapped.Code)
	}
	if wrapped.Message != "connection refused" {
		t.Errorf("expected message 'connection refused', got '%s'", wrapped.Message)
	}
}

func TestErrorCodes(t *testing.T) {
	codes := []ErrorCode{
		ErrNotFound,
		ErrAlreadyExists,
		ErrTimeout,
		ErrPermission,
		ErrInvalidInput,
		ErrK8sConnection,
		ErrInternal,
	}

	for _, code := range codes {
		if code == "" {
			t.Error("error code should not be empty")
		}
	}
}
