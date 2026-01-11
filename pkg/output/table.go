package output

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

// TablePrinter prints output in table format.
type TablePrinter struct {
	writer *tabwriter.Writer
}

// NewTablePrinter creates a new table printer.
func NewTablePrinter(w io.Writer) *TablePrinter {
	return &TablePrinter{
		writer: tabwriter.NewWriter(w, 0, 0, 2, ' ', 0),
	}
}

// DefaultTablePrinter returns a table printer writing to stdout.
func DefaultTablePrinter() *TablePrinter {
	return NewTablePrinter(os.Stdout)
}

// PrintHeader prints the table header.
func (p *TablePrinter) PrintHeader(headers ...string) {
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(p.writer, "\t")
		}
		fmt.Fprint(p.writer, h)
	}
	fmt.Fprintln(p.writer)
}

// PrintRow prints a table row.
func (p *TablePrinter) PrintRow(values ...interface{}) {
	for i, v := range values {
		if i > 0 {
			fmt.Fprint(p.writer, "\t")
		}
		fmt.Fprint(p.writer, v)
	}
	fmt.Fprintln(p.writer)
}

// Flush flushes the table writer.
func (p *TablePrinter) Flush() error {
	return p.writer.Flush()
}

// PrintTable prints a complete table with headers and rows.
func (p *TablePrinter) PrintTable(headers []string, rows [][]string) error {
	p.PrintHeader(headers...)
	for _, row := range rows {
		values := make([]interface{}, len(row))
		for i, v := range row {
			values[i] = v
		}
		p.PrintRow(values...)
	}
	return p.Flush()
}

// Writer returns the underlying tabwriter for custom formatting.
func (p *TablePrinter) Writer() *tabwriter.Writer {
	return p.writer
}
