package output

import (
	"encoding/json"
	"io"
	"os"
)

// JSONPrinter prints output in JSON format.
type JSONPrinter struct {
	writer io.Writer
	indent bool
}

// NewJSONPrinter creates a new JSON printer.
func NewJSONPrinter(w io.Writer) *JSONPrinter {
	return &JSONPrinter{
		writer: w,
		indent: true,
	}
}

// NewJSONPrinterNoIndent creates a new JSON printer without indentation.
func NewJSONPrinterNoIndent(w io.Writer) *JSONPrinter {
	return &JSONPrinter{
		writer: w,
		indent: false,
	}
}

// Print prints the data as JSON.
func (p *JSONPrinter) Print(data interface{}) error {
	encoder := json.NewEncoder(p.writer)
	if p.indent {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(data)
}

// PrintResult prints a Result as JSON.
func (p *JSONPrinter) PrintResult(result *Result) error {
	return p.Print(result)
}

// PrintSuccess prints a success result with data.
func (p *JSONPrinter) PrintSuccess(data interface{}) error {
	return p.PrintResult(NewSuccessResult(data))
}

// PrintSuccessMessage prints a success result with a message.
func (p *JSONPrinter) PrintSuccessMessage(message string, data interface{}) error {
	return p.PrintResult(NewSuccessResultWithMessage(message, data))
}

// PrintError prints an error result.
func (p *JSONPrinter) PrintError(err error) error {
	var structuredErr *StructuredError
	if se, ok := err.(*StructuredError); ok {
		structuredErr = se
	} else {
		structuredErr = &StructuredError{
			Code:    ErrInternal,
			Message: err.Error(),
		}
	}
	return p.PrintResult(NewErrorResult(structuredErr))
}

// DefaultJSONPrinter returns a JSON printer writing to stdout.
func DefaultJSONPrinter() *JSONPrinter {
	return NewJSONPrinter(os.Stdout)
}
