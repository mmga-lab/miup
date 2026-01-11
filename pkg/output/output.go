package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Format represents the output format.
type Format string

const (
	FormatHuman Format = "human"
	FormatJSON  Format = "json"
)

// Result represents a unified result structure for all commands.
type Result struct {
	Success bool             `json:"success"`
	Message string           `json:"message,omitempty"`
	Data    interface{}      `json:"data,omitempty"`
	Error   *StructuredError `json:"error,omitempty"`
}

// NewSuccessResult creates a success result with data.
func NewSuccessResult(data interface{}) *Result {
	return &Result{
		Success: true,
		Data:    data,
	}
}

// NewSuccessResultWithMessage creates a success result with a message.
func NewSuccessResultWithMessage(message string, data interface{}) *Result {
	return &Result{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// NewErrorResult creates an error result.
func NewErrorResult(err *StructuredError) *Result {
	return &Result{
		Success: false,
		Error:   err,
	}
}

// PrintJSON prints the result as JSON to the given writer.
func PrintJSON(w io.Writer, result *Result) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// PrintErrorJSON prints an error as JSON to the given writer.
func PrintErrorJSON(w io.Writer, err error) error {
	var structuredErr *StructuredError
	if se, ok := err.(*StructuredError); ok {
		structuredErr = se
	} else {
		structuredErr = &StructuredError{
			Code:    ErrInternal,
			Message: err.Error(),
		}
	}

	result := NewErrorResult(structuredErr)
	return PrintJSON(w, result)
}

// PrintDataJSON prints data directly as JSON (for backward compatibility with existing --json flags).
func PrintDataJSON(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// MustPrintJSON prints JSON and exits on error.
func MustPrintJSON(result *Result) {
	if err := PrintJSON(os.Stdout, result); err != nil {
		fmt.Fprintf(os.Stderr, "failed to print JSON: %v\n", err)
		os.Exit(1)
	}
}

// MustPrintDataJSON prints data as JSON and exits on error.
func MustPrintDataJSON(data interface{}) {
	if err := PrintDataJSON(os.Stdout, data); err != nil {
		fmt.Fprintf(os.Stderr, "failed to print JSON: %v\n", err)
		os.Exit(1)
	}
}
