package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ishk9/flog/internal/filter"
	"github.com/ishk9/flog/internal/parser"
)

func TestIntegration_FilterJSONLogs(t *testing.T) {
	// Create temp log file
	content := `{"timestamp":"2024-01-15T10:00:00Z","level":"info","message":"Application started"}
{"timestamp":"2024-01-15T10:00:01Z","level":"error","message":"Connection failed","status":500}
{"timestamp":"2024-01-15T10:00:02Z","level":"warn","message":"High memory usage"}
{"timestamp":"2024-01-15T10:00:03Z","level":"error","message":"Timeout occurred","status":504}
{"timestamp":"2024-01-15T10:00:04Z","level":"info","message":"Request completed","status":200}`

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{
			name:      "filter by level:error",
			query:     "level:error",
			wantCount: 2,
		},
		{
			name:      "filter by level:info",
			query:     "level:info",
			wantCount: 2,
		},
		{
			name:      "filter by status>=500",
			query:     "status>=500",
			wantCount: 2,
		},
		{
			name:      "filter by status:200",
			query:     "status:200",
			wantCount: 1,
		},
		{
			name:      "filter by level:error|level:warn",
			query:     "level:error|level:warn",
			wantCount: 3,
		},
		{
			name:      "filter by level:error,status>=500",
			query:     "level:error,status>=500",
			wantCount: 2,
		},
		{
			name:      "regex filter",
			query:     "message~=failed",
			wantCount: 1,
		},
		{
			name:      "contains filter",
			query:     "message*=memory",
			wantCount: 1,
		},
		{
			name:      "exists filter",
			query:     "status?",
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse query
			qp := filter.NewQueryParser()
			chain, err := qp.Parse(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			// Read and filter
			reader := parser.NewStreamReader()
			p := parser.NewAutoParser()
			matcher := filter.NewMatcher(false)

			ctx := context.Background()
			lines, errs := reader.ReadLines(ctx, logFile)

			count := 0
			lineNum := 0

			for line := range lines {
				lineNum++
				entry, err := p.Parse(line, lineNum)
				if err != nil {
					continue
				}

				if matcher.Match(entry, chain) {
					count++
				}
				parser.ReleaseEntry(entry)
			}

			// Check for errors
			select {
			case err := <-errs:
				if err != nil {
					t.Fatalf("Read error: %v", err)
				}
			default:
			}

			if count != tt.wantCount {
				t.Errorf("Filter matched %d lines, want %d", count, tt.wantCount)
			}
		})
	}
}

func TestIntegration_FilterKeyValueLogs(t *testing.T) {
	content := `timestamp=2024-01-15T10:00:00Z level=info message="Application started"
timestamp=2024-01-15T10:00:01Z level=error message="Connection failed" status=500
timestamp=2024-01-15T10:00:02Z level=warn message="High memory usage" memory=85.5
timestamp=2024-01-15T10:00:03Z level=error message="Timeout occurred" status=504
timestamp=2024-01-15T10:00:04Z level=info message="Request completed" status=200`

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{
			name:      "filter by level:error",
			query:     "level:error",
			wantCount: 2,
		},
		{
			name:      "filter by status>=500",
			query:     "status>=500",
			wantCount: 2,
		},
		{
			name:      "filter by memory exists",
			query:     "memory?",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qp := filter.NewQueryParser()
			chain, err := qp.Parse(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			reader := parser.NewStreamReader()
			p := parser.NewAutoParser()
			matcher := filter.NewMatcher(false)

			ctx := context.Background()
			lines, errs := reader.ReadLines(ctx, logFile)

			count := 0
			lineNum := 0

			for line := range lines {
				lineNum++
				entry, err := p.Parse(line, lineNum)
				if err != nil {
					continue
				}

				if matcher.Match(entry, chain) {
					count++
				}
				parser.ReleaseEntry(entry)
			}

			select {
			case err := <-errs:
				if err != nil {
					t.Fatalf("Read error: %v", err)
				}
			default:
			}

			if count != tt.wantCount {
				t.Errorf("Filter matched %d lines, want %d", count, tt.wantCount)
			}
		})
	}
}

func TestIntegration_NestedFields(t *testing.T) {
	content := `{"user":{"id":123,"profile":{"role":"admin"}},"action":"login"}
{"user":{"id":456,"profile":{"role":"user"}},"action":"view"}
{"user":{"id":123,"profile":{"role":"admin"}},"action":"delete"}
{"user":{"id":789,"profile":{"role":"user"}},"action":"login"}`

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{
			name:      "filter by user.id:123",
			query:     "user.id:123",
			wantCount: 2,
		},
		{
			name:      "filter by user.profile.role:admin",
			query:     "user.profile.role:admin",
			wantCount: 2,
		},
		{
			name:      "filter by role:admin AND action:delete",
			query:     "user.profile.role:admin,action:delete",
			wantCount: 1,
		},
		{
			name:      "filter by role:user OR action:login",
			query:     "user.profile.role:user|action:login",
			wantCount: 3, // 2 users + 2 logins with 1 overlap
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qp := filter.NewQueryParser()
			chain, err := qp.Parse(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			reader := parser.NewStreamReader()
			p := parser.NewAutoParser()
			matcher := filter.NewMatcher(false)

			ctx := context.Background()
			lines, errs := reader.ReadLines(ctx, logFile)

			count := 0
			lineNum := 0

			for line := range lines {
				lineNum++
				entry, err := p.Parse(line, lineNum)
				if err != nil {
					continue
				}

				if matcher.Match(entry, chain) {
					count++
				}
				parser.ReleaseEntry(entry)
			}

			select {
			case err := <-errs:
				if err != nil {
					t.Fatalf("Read error: %v", err)
				}
			default:
			}

			if count != tt.wantCount {
				t.Errorf("Filter matched %d lines, want %d", count, tt.wantCount)
			}
		})
	}
}

func TestIntegration_CaseInsensitive(t *testing.T) {
	content := `{"level":"ERROR","message":"Something went wrong"}
{"level":"Error","message":"Another error"}
{"level":"error","message":"Yet another error"}
{"level":"INFO","message":"All good"}`

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	qp := filter.NewQueryParser()
	chain, _ := qp.Parse("level:error")

	reader := parser.NewStreamReader()
	p := parser.NewAutoParser()
	matcher := filter.NewMatcher(true) // Case insensitive

	ctx := context.Background()
	lines, _ := reader.ReadLines(ctx, logFile)

	count := 0
	lineNum := 0

	for line := range lines {
		lineNum++
		entry, err := p.Parse(line, lineNum)
		if err != nil {
			continue
		}

		if matcher.Match(entry, chain) {
			count++
		}
		parser.ReleaseEntry(entry)
	}

	if count != 3 {
		t.Errorf("Case insensitive filter matched %d lines, want 3", count)
	}
}

// Benchmark for full pipeline
func BenchmarkIntegration_Pipeline(b *testing.B) {
	// Create temp file with 1000 lines
	var content string
	for i := 0; i < 1000; i++ {
		level := "info"
		if i%10 == 0 {
			level = "error"
		}
		content += `{"timestamp":"2024-01-15T10:00:00Z","level":"` + level + `","status":` + "500" + `,"message":"Request processed"}` + "\n"
	}

	tmpDir := b.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	os.WriteFile(logFile, []byte(content), 0644)

	qp := filter.NewQueryParser()
	chain, _ := qp.Parse("level:error,status>=400")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := parser.NewStreamReader()
		p := parser.NewAutoParser()
		matcher := filter.NewMatcher(false)

		ctx := context.Background()
		lines, _ := reader.ReadLines(ctx, logFile)

		lineNum := 0
		for line := range lines {
			lineNum++
			entry, err := p.Parse(line, lineNum)
			if err != nil {
				continue
			}
			matcher.Match(entry, chain)
			parser.ReleaseEntry(entry)
		}
	}
}

