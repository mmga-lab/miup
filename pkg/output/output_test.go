package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestNewSuccessResult(t *testing.T) {
	data := map[string]string{"key": "value"}
	result := NewSuccessResult(data)

	if !result.Success {
		t.Error("expected success to be true")
	}
	if result.Error != nil {
		t.Error("expected error to be nil")
	}
	if result.Data == nil {
		t.Error("expected data to be set")
	}
}

func TestNewSuccessResultWithMessage(t *testing.T) {
	result := NewSuccessResultWithMessage("test message", nil)

	if !result.Success {
		t.Error("expected success to be true")
	}
	if result.Message != "test message" {
		t.Errorf("expected message 'test message', got '%s'", result.Message)
	}
}

func TestNewErrorResult(t *testing.T) {
	err := NewError(ErrNotFound, "resource not found")
	result := NewErrorResult(err)

	if result.Success {
		t.Error("expected success to be false")
	}
	if result.Error == nil {
		t.Error("expected error to be set")
	}
	if result.Error.Code != ErrNotFound {
		t.Errorf("expected code %s, got %s", ErrNotFound, result.Error.Code)
	}
}

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	result := NewSuccessResult(map[string]int{"count": 42})

	err := PrintJSON(&buf, result)
	if err != nil {
		t.Fatalf("PrintJSON failed: %v", err)
	}

	var parsed Result
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if !parsed.Success {
		t.Error("expected success to be true in parsed result")
	}
}

func TestPrintErrorJSON(t *testing.T) {
	var buf bytes.Buffer
	err := errors.New("something went wrong")

	if err := PrintErrorJSON(&buf, err); err != nil {
		t.Fatalf("PrintErrorJSON failed: %v", err)
	}

	var result Result
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result.Success {
		t.Error("expected success to be false")
	}
	if result.Error == nil {
		t.Error("expected error to be set")
	}
	if result.Error.Code != ErrInternal {
		t.Errorf("expected code %s, got %s", ErrInternal, result.Error.Code)
	}
}

func TestPrintErrorJSON_StructuredError(t *testing.T) {
	var buf bytes.Buffer
	structuredErr := NewError(ErrNotFound, "user not found")

	if err := PrintErrorJSON(&buf, structuredErr); err != nil {
		t.Fatalf("PrintErrorJSON failed: %v", err)
	}

	var result Result
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result.Error.Code != ErrNotFound {
		t.Errorf("expected code %s, got %s", ErrNotFound, result.Error.Code)
	}
}
