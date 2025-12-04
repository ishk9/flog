package tests

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ishk9/flog/internal/output"
	"github.com/ishk9/flog/internal/parser"
)

func TestRawFormatter(t *testing.T) {
	formatter := output.NewRawFormatter()

	entry := parser.NewLogEntry(`{"level":"error","status":500}`, 1)
	entry.Fields["level"] = "error"

	result := formatter.Format(entry)
	if result != `{"level":"error","status":500}` {
		t.Errorf("RawFormatter.Format() = %v, want original line", result)
	}
}

func TestJSONFormatter(t *testing.T) {
	formatter := output.NewJSONFormatter()

	tests := []struct {
		name    string
		entry   *parser.LogEntry
		wantJSON bool
	}{
		{
			name: "JSON input stays JSON",
			entry: func() *parser.LogEntry {
				e := parser.NewLogEntry(`{"level":"error"}`, 1)
				e.Fields["level"] = "error"
				return e
			}(),
			wantJSON: true,
		},
		{
			name: "non-JSON gets converted",
			entry: func() *parser.LogEntry {
				e := parser.NewLogEntry(`level=error`, 1)
				e.Fields["level"] = "error"
				return e
			}(),
			wantJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.Format(tt.entry)
			if tt.wantJSON && !strings.HasPrefix(result, "{") {
				t.Errorf("JSONFormatter.Format() = %v, want JSON", result)
			}
		})
	}
}

func TestPrettyFormatter(t *testing.T) {
	formatter := output.NewPrettyFormatter(false) // No colors for testing

	entry := parser.NewLogEntry(`{"level":"error","status":500}`, 1)
	entry.Fields["level"] = "error"
	entry.Fields["status"] = float64(500)

	result := formatter.Format(entry)

	// Should be indented
	if !strings.Contains(result, "\n") {
		t.Error("PrettyFormatter should produce indented output")
	}

	// Should contain field names
	if !strings.Contains(result, "level") {
		t.Error("PrettyFormatter should include field names")
	}
}

func TestFieldsFormatter(t *testing.T) {
	entry := parser.NewLogEntry(`{"timestamp":"2024-01-15","level":"error","status":500}`, 1)
	entry.Fields["timestamp"] = "2024-01-15"
	entry.Fields["level"] = "error"
	entry.Fields["status"] = float64(500)

	tests := []struct {
		name     string
		fields   []string
		useJSON  bool
		contains []string
	}{
		{
			name:     "tab separated",
			fields:   []string{"timestamp", "level"},
			useJSON:  false,
			contains: []string{"2024-01-15", "error"},
		},
		{
			name:     "JSON format",
			fields:   []string{"timestamp", "level"},
			useJSON:  true,
			contains: []string{`"timestamp"`, `"level"`},
		},
		{
			name:     "missing field shows dash",
			fields:   []string{"nonexistent"},
			useJSON:  false,
			contains: []string{"-"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := output.NewFieldsFormatter(tt.fields, tt.useJSON)
			result := formatter.Format(entry)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("FieldsFormatter.Format() = %v, want to contain %v", result, want)
				}
			}
		})
	}
}

func TestWriter(t *testing.T) {
	var buf bytes.Buffer
	formatter := output.NewRawFormatter()
	stats := output.NewStats()
	writer := output.NewWriter(&buf, formatter, stats)

	entry := parser.NewLogEntry(`{"level":"error"}`, 1)

	// Write entry
	ok := writer.Write(entry)
	if !ok {
		t.Error("Write() should return true")
	}
	if writer.Count() != 1 {
		t.Errorf("Count() = %v, want 1", writer.Count())
	}

	// Check output
	if !strings.Contains(buf.String(), "error") {
		t.Error("Writer should write formatted output")
	}
}

func TestWriter_WithLimit(t *testing.T) {
	var buf bytes.Buffer
	formatter := output.NewRawFormatter()
	stats := output.NewStats()
	writer := output.NewWriter(&buf, formatter, stats)
	writer.SetLimit(2)

	entry := parser.NewLogEntry(`{"level":"error"}`, 1)

	// Write within limit
	ok := writer.Write(entry)
	if !ok {
		t.Error("Write() should return true within limit")
	}

	ok = writer.Write(entry)
	if ok {
		t.Error("Write() should return false at limit")
	}

	// Third write should fail
	ok = writer.Write(entry)
	if ok {
		t.Error("Write() should return false beyond limit")
	}

	if writer.Count() != 2 {
		t.Errorf("Count() = %v, want 2", writer.Count())
	}
}

func TestStats(t *testing.T) {
	stats := output.NewStats()

	stats.IncrTotal()
	stats.IncrTotal()
	stats.IncrMatched()
	stats.IncrErrors()

	if stats.TotalLines != 2 {
		t.Errorf("TotalLines = %v, want 2", stats.TotalLines)
	}
	if stats.MatchedLines != 1 {
		t.Errorf("MatchedLines = %v, want 1", stats.MatchedLines)
	}
	if stats.ParseErrors != 1 {
		t.Errorf("ParseErrors = %v, want 1", stats.ParseErrors)
	}

	stats.Finish()
	if stats.Duration == 0 {
		t.Error("Duration should be set after Finish()")
	}
}

// Benchmarks

func BenchmarkRawFormatter(b *testing.B) {
	formatter := output.NewRawFormatter()
	entry := parser.NewLogEntry(`{"level":"error","status":500}`, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format(entry)
	}
}

func BenchmarkPrettyFormatter(b *testing.B) {
	formatter := output.NewPrettyFormatter(false)
	entry := parser.NewLogEntry(`{"level":"error","status":500}`, 1)
	entry.Fields["level"] = "error"
	entry.Fields["status"] = float64(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format(entry)
	}
}

func BenchmarkFieldsFormatter(b *testing.B) {
	formatter := output.NewFieldsFormatter([]string{"level", "status"}, false)
	entry := parser.NewLogEntry(`{"level":"error","status":500}`, 1)
	entry.Fields["level"] = "error"
	entry.Fields["status"] = float64(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format(entry)
	}
}

