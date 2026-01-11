package output

// ErrorCode represents structured error codes for agent-friendly error handling.
type ErrorCode string

const (
	ErrNotFound      ErrorCode = "NOT_FOUND"
	ErrAlreadyExists ErrorCode = "ALREADY_EXISTS"
	ErrTimeout       ErrorCode = "TIMEOUT"
	ErrPermission    ErrorCode = "PERMISSION_DENIED"
	ErrInvalidInput  ErrorCode = "INVALID_INPUT"
	ErrK8sConnection ErrorCode = "K8S_CONNECTION_ERROR"
	ErrInternal      ErrorCode = "INTERNAL_ERROR"
)

// StructuredError represents an error with a code and message for JSON output.
type StructuredError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *StructuredError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// NewError creates a new StructuredError.
func NewError(code ErrorCode, message string) *StructuredError {
	return &StructuredError{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithDetails creates a new StructuredError with details.
func NewErrorWithDetails(code ErrorCode, message, details string) *StructuredError {
	return &StructuredError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// WrapError wraps an existing error with a code.
func WrapError(code ErrorCode, err error) *StructuredError {
	return &StructuredError{
		Code:    code,
		Message: err.Error(),
	}
}
