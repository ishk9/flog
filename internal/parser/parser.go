// Package parser provides log parsing functionality for various formats.
package parser

import "sync"

// LogEntry represents a parsed log line with extracted fields.
type LogEntry struct {
	Raw     string         // Original log line
	Fields  map[string]any // Flattened key-value fields
	LineNum int            // Line number in source file
}

// Parser defines the interface for log format parsers.
type Parser interface {
	// Parse converts a raw log line into a structured LogEntry.
	Parse(line string, lineNum int) (*LogEntry, error)

	// CanParse checks if this parser can handle the given line format.
	CanParse(line string) bool
}

// entryPool provides object pooling for LogEntry to reduce GC pressure.
var entryPool = sync.Pool{
	New: func() any {
		return &LogEntry{
			Fields: make(map[string]any, 16),
		}
	},
}

// AcquireEntry gets a LogEntry from the pool.
func AcquireEntry() *LogEntry {
	entry := entryPool.Get().(*LogEntry)
	// Clear existing fields
	for k := range entry.Fields {
		delete(entry.Fields, k)
	}
	entry.Raw = ""
	entry.LineNum = 0
	return entry
}

// ReleaseEntry returns a LogEntry to the pool.
func ReleaseEntry(entry *LogEntry) {
	if entry != nil {
		entryPool.Put(entry)
	}
}

// NewLogEntry creates a new LogEntry with initialized fields map.
// Use AcquireEntry for high-performance scenarios.
func NewLogEntry(line string, lineNum int) *LogEntry {
	return &LogEntry{
		Raw:     line,
		Fields:  make(map[string]any, 16),
		LineNum: lineNum,
	}
}
