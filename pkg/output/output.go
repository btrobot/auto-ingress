package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
)

// Output handles formatted terminal output
type Output struct {
	writer   io.Writer
	jsonMode bool
	noColor  bool
}

// New creates an Output writing to the given writer
func New(w io.Writer) *Output {
	return &Output{writer: w}
}

// Stdout returns an Output writing to stdout
func Stdout() *Output { return New(os.Stdout) }

// SetJSONMode enables/disables JSON output
func (o *Output) SetJSONMode(enabled bool) { o.jsonMode = enabled }

// SetNoColor enables/disables colored output
func (o *Output) SetNoColor(enabled bool) { o.noColor = enabled }

// Success prints a green success message
func (o *Output) Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if o.noColor {
		fmt.Fprintf(o.writer, "✓ %s\n", msg)
	} else {
		fmt.Fprintf(o.writer, "%s✓ %s%s\n", ColorGreen, msg, ColorReset)
	}
}

// Error prints a red error message
func (o *Output) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if o.noColor {
		fmt.Fprintf(o.writer, "✗ %s\n", msg)
	} else {
		fmt.Fprintf(o.writer, "%s✗ %s%s\n", ColorRed, msg, ColorReset)
	}
}

// Warning prints a yellow warning message
func (o *Output) Warning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if o.noColor {
		fmt.Fprintf(o.writer, "⚠ %s\n", msg)
	} else {
		fmt.Fprintf(o.writer, "%s⚠ %s%s\n", ColorYellow, msg, ColorReset)
	}
}

// Info prints a blue info message
func (o *Output) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if o.noColor {
		fmt.Fprintf(o.writer, "ℹ %s\n", msg)
	} else {
		fmt.Fprintf(o.writer, "%sℹ %s%s\n", ColorBlue, msg, ColorReset)
	}
}

// Header prints a bold header
func (o *Output) Header(text string) {
	if o.noColor {
		fmt.Fprintf(o.writer, "%s\n", text)
	} else {
		fmt.Fprintf(o.writer, "%s%s%s\n", ColorBold, text, ColorReset)
	}
}

// Dim prints dimmed text
func (o *Output) Dim(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if o.noColor {
		fmt.Fprintf(o.writer, "%s\n", msg)
	} else {
		fmt.Fprintf(o.writer, "%s%s%s\n", ColorDim, msg, ColorReset)
	}
}

// Table renders a table with headers and rows
func (o *Output) Table(headers []string, rows [][]string) {
	if o.jsonMode {
		var data []map[string]string
		for _, row := range rows {
			m := make(map[string]string)
			for i, h := range headers {
				if i < len(row) {
					m[strings.ToLower(h)] = row[i]
				}
			}
			data = append(data, m)
		}
		enc := json.NewEncoder(o.writer)
		enc.SetIndent("", "  ")
		enc.Encode(data)
		return
	}

	tw := tabwriter.NewWriter(o.writer, 0, 0, 2, ' ', 0)
	// Header
	if !o.noColor {
		fmt.Fprintf(tw, "%s%s%s\n", ColorBold, strings.Join(headers, "\t"), ColorReset)
	} else {
		fmt.Fprintf(tw, "%s\n", strings.Join(headers, "\t"))
	}
	// Rows
	for _, row := range rows {
		fmt.Fprintf(tw, "%s\n", strings.Join(row, "\t"))
	}
	tw.Flush()
}

// StatusIndicator returns a colored status dot
func StatusIndicator(status string) string {
	return StatusIndicatorWithColor(status, false)
}

// StatusIndicatorWithColor returns a status dot, respecting noColor
func StatusIndicatorWithColor(status string, noColor bool) string {
	dot := "●"
	if noColor {
		switch status {
		case "running", "healthy", "valid":
			return dot
		case "pending", "starting":
			return dot
		default:
			return dot
		}
	}
	switch status {
	case "running", "healthy", "valid":
		return ColorGreen + dot + ColorReset
	case "pending", "starting":
		return ColorYellow + dot + ColorReset
	case "stopped", "expired", "error", "dead":
		return ColorRed + dot + ColorReset
	default:
		return ColorDim + dot + ColorReset
	}
}

// JSON outputs data as formatted JSON
func (o *Output) JSON(data interface{}) error {
	enc := json.NewEncoder(o.writer)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// KeyValue prints a key-value pair
func (o *Output) KeyValue(key, value string) {
	fmt.Fprintf(o.writer, "  %-14s %s\n", key+":", value)
}
