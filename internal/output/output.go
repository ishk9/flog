// Package output provides formatting and display functionality for filtered logs.
package output

import "github.com/ishk9/flog/internal/parser"

// Mode represents the output mode for filtered results.
type Mode int

const (
	ModeLines Mode = iota // Print matching lines
	ModeCount             // Print count only
	ModeStats             // Print field statistics
	ModeFirst             // Print first N matches
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
}

// NewStats creates a new Stats instance with initialized maps.
func NewStats() *Stats {
	return &Stats{
		FieldCounts: make(map[string]int64),
	}
}

