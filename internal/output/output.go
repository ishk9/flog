// Package output provides formatting and display functionality for filtered logs.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ishk9/flog/internal/parser"
)

// Mode represents the output mode for filtered results.
type Mode int

const (
	ModeLines Mode = iota // Print matching lines
	ModeCount             // Print count only
	ModeStats             // Print field statistics
	ModeFirst             // Print first N matches
)

// Format represents the output format.
type Format int

const (
	FormatRaw    Format = iota // Original log line
	FormatPretty               // Pretty-printed JSON
	FormatJSON                 // Compact JSON
	FormatFields               // Selected fields only
)

// Formatter defines the interface for output formatting.
type Formatter interface {
	// Format converts a log entry to a displayable string.
	Format(entry *parser.LogEntry) string
}

// Stats holds statistics about the filtering operation.
type Stats struct {
	TotalLines   int64            // Total lines processed
	MatchedLines int64            // Lines that matched filters
	ParseErrors  int64            // Lines that failed to parse
	FieldCounts  map[string]int64 // Field occurrence counts (for --stats)
	StartTime    time.Time        // When processing started
	Duration     time.Duration    // Total processing time
}

// NewStats creates a new Stats instance with initialized maps.
func NewStats() *Stats {
	return &Stats{
		FieldCounts: make(map[string]int64),
		StartTime:   time.Now(),
	}
}

// IncrTotal atomically increments the total line count.
func (s *Stats) IncrTotal() {
	atomic.AddInt64(&s.TotalLines, 1)
}

// IncrMatched atomically increments the matched line count.
func (s *Stats) IncrMatched() {
	atomic.AddInt64(&s.MatchedLines, 1)
}

// IncrErrors atomically increments the parse error count.
func (s *Stats) IncrErrors() {
	atomic.AddInt64(&s.ParseErrors, 1)
}

// Finish marks the stats as complete.
func (s *Stats) Finish() {
	s.Duration = time.Since(s.StartTime)
}

// RawFormatter outputs the original log line.
type RawFormatter struct{}

// NewRawFormatter creates a new raw formatter.
func NewRawFormatter() *RawFormatter {
	return &RawFormatter{}
}

// Format returns the original log line.
func (f *RawFormatter) Format(entry *parser.LogEntry) string {
	return entry.Raw
}

// PrettyFormatter outputs pretty-printed JSON with colors.
type PrettyFormatter struct {
	Indent    string
	UseColors bool
}

// NewPrettyFormatter creates a new pretty formatter.
func NewPrettyFormatter(useColors bool) *PrettyFormatter {
	return &PrettyFormatter{
		Indent:    "  ",
		UseColors: useColors,
	}
}

// Format returns pretty-printed JSON.
func (f *PrettyFormatter) Format(entry *parser.LogEntry) string {
	// Rebuild nested structure from flattened fields
	nested := unflattenMap(entry.Fields)

	data, err := json.MarshalIndent(nested, "", f.Indent)
	if err != nil {
		return entry.Raw
	}

	if f.UseColors {
		return colorizeJSON(string(data))
	}
	return string(data)
}

// JSONFormatter outputs compact JSON.
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSON formatter.
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// Format returns compact JSON.
func (f *JSONFormatter) Format(entry *parser.LogEntry) string {
	// If the original was valid JSON, just return it
	if strings.HasPrefix(strings.TrimSpace(entry.Raw), "{") {
		return strings.TrimSpace(entry.Raw)
	}

	// Otherwise, serialize fields
	nested := unflattenMap(entry.Fields)
	data, err := json.Marshal(nested)
	if err != nil {
		return entry.Raw
	}
	return string(data)
}

// FieldsFormatter outputs only selected fields.
type FieldsFormatter struct {
	Fields    []string
	Separator string
	UseJSON   bool
}

// NewFieldsFormatter creates a new fields formatter.
func NewFieldsFormatter(fields []string, useJSON bool) *FieldsFormatter {
	return &FieldsFormatter{
		Fields:    fields,
		Separator: "\t",
		UseJSON:   useJSON,
	}
}

// Format returns only the selected fields.
func (f *FieldsFormatter) Format(entry *parser.LogEntry) string {
	if f.UseJSON {
		result := make(map[string]any)
		for _, field := range f.Fields {
			if val, ok := entry.Fields[field]; ok {
				result[field] = val
			}
		}
		data, _ := json.Marshal(result)
		return string(data)
	}

	var parts []string
	for _, field := range f.Fields {
		if val, ok := entry.Fields[field]; ok {
			parts = append(parts, fmt.Sprintf("%v", val))
		} else {
			parts = append(parts, "-")
		}
	}
	return strings.Join(parts, f.Separator)
}

// Writer handles writing formatted output.
type Writer struct {
	out       io.Writer
	formatter Formatter
	stats     *Stats
	limit     int64
	count     int64
}

// NewWriter creates a new output writer.
func NewWriter(out io.Writer, formatter Formatter, stats *Stats) *Writer {
	return &Writer{
		out:       out,
		formatter: formatter,
		stats:     stats,
		limit:     -1, // No limit
	}
}

// SetLimit sets the maximum number of entries to write.
func (w *Writer) SetLimit(n int64) {
	w.limit = n
}

// Write writes a single entry to output.
// Returns true if the entry was written, false if limit reached.
func (w *Writer) Write(entry *parser.LogEntry) bool {
	if w.limit >= 0 && w.count >= w.limit {
		return false
	}

	formatted := w.formatter.Format(entry)
	fmt.Fprintln(w.out, formatted)

	w.count++
	if w.stats != nil {
		w.stats.IncrMatched()
	}

	return w.limit < 0 || w.count < w.limit
}

// Count returns the number of entries written.
func (w *Writer) Count() int64 {
	return w.count
}

// unflattenMap converts a flattened map back to nested structure.
func unflattenMap(flat map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range flat {
		// Skip nested objects that were stored alongside flattened keys
		if _, isMap := value.(map[string]any); isMap {
			continue
		}

		parts := strings.Split(key, ".")
		current := result

		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = value
			} else {
				if _, exists := current[part]; !exists {
					current[part] = make(map[string]any)
				}
				if next, ok := current[part].(map[string]any); ok {
					current = next
				} else {
					// Conflict - just store the value directly
					current[part] = value
					break
				}
			}
		}
	}

	return result
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

// colorizeJSON adds ANSI colors to JSON output.
func colorizeJSON(s string) string {
	var result strings.Builder
	inString := false
	inKey := false
	n := len(s)

	for i := 0; i < n; i++ {
		ch := s[i]

		switch {
		case ch == '"' && (i == 0 || s[i-1] != '\\'):
			if inString {
				result.WriteByte(ch)
				result.WriteString(colorReset)
				inString = false
				_ = inKey // Mark as used
			} else {
				inString = true
				// Check if this is a key (followed by ':')
				j := i + 1
				for j < n && s[j] != '"' {
					if s[j] == '\\' && j+1 < n {
						j++
					}
					j++
				}
				if j+1 < n && s[j+1] == ':' {
					inKey = true
					result.WriteString(colorCyan)
				} else {
					result.WriteString(colorGreen)
				}
				result.WriteByte(ch)
			}
		case !inString && (ch == '{' || ch == '}' || ch == '[' || ch == ']'):
			result.WriteString(colorYellow)
			result.WriteByte(ch)
			result.WriteString(colorReset)
		case !inString && ch == ':':
			result.WriteByte(ch)
		case !inString && (ch >= '0' && ch <= '9' || ch == '-'):
			result.WriteString(colorBlue)
			// Collect the whole number
			for i < n && (s[i] >= '0' && s[i] <= '9' || s[i] == '-' || s[i] == '.' || s[i] == 'e' || s[i] == 'E' || s[i] == '+') {
				result.WriteByte(s[i])
				i++
			}
			i--
			result.WriteString(colorReset)
		case !inString && i+4 <= n && (s[i:i+4] == "true" || s[i:i+4] == "null"):
			result.WriteString(colorRed)
			result.WriteString(s[i : i+4])
			result.WriteString(colorReset)
			i += 3
		case !inString && i+5 <= n && s[i:i+5] == "false":
			result.WriteString(colorRed)
			result.WriteString("false")
			result.WriteString(colorReset)
			i += 4
		default:
			result.WriteByte(ch)
		}
	}

	return result.String()
}
