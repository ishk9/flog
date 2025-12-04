// Package parser provides log parsing functionality for various formats.
package parser

// LogEntry represents a parsed log line with extracted fields.
type LogEntry struct {
	Raw     string         // Original log line
	Fields  map[string]any // Flattened key-value fields
	LineNum int            // Line number in source file
}

// Parser defines the interface for log format parsers.
type Parser interface {
	// Parse converts a raw log line into a structured LogEntry.
	Parse(line string) (*LogEntry, error)

	// CanParse checks if this parser can handle the given line format.
	CanParse(line string) bool
}

// NewLogEntry creates a new LogEntry with initialized fields map.
func NewLogEntry(line string, lineNum int) *LogEntry {
	return &LogEntry{
		Raw:     line,
		Fields:  make(map[string]any),
		LineNum: lineNum,
	}
}

