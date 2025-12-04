package tests

import (
	"testing"

	"github.com/ishk9/flog/internal/parser"
)

func TestJSONParser_Parse(t *testing.T) {
	p := parser.NewJSONParser()

	tests := []struct {
		name     string
		line     string
		wantErr  bool
		checkFn  func(*parser.LogEntry) bool
	}{
		{
			name:    "simple JSON",
			line:    `{"level":"error","status":500}`,
			wantErr: false,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["level"] == "error" && e.Fields["status"] == float64(500)
			},
		},
		{
			name:    "nested JSON",
			line:    `{"user":{"id":123,"name":"john"}}`,
			wantErr: false,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["user.id"] == float64(123) && e.Fields["user.name"] == "john"
			},
		},
		{
			name:    "deeply nested JSON",
			line:    `{"user":{"profile":{"role":"admin"}}}`,
			wantErr: false,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["user.profile.role"] == "admin"
			},
		},
		{
			name:    "array values",
			line:    `{"tags":["a","b","c"],"count":3}`,
			wantErr: false,
			checkFn: func(e *parser.LogEntry) bool {
				tags, ok := e.Fields["tags"].([]any)
				return ok && len(tags) == 3
			},
		},
		{
			name:    "boolean values",
			line:    `{"active":true,"deleted":false}`,
			wantErr: false,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["active"] == true && e.Fields["deleted"] == false
			},
		},
		{
			name:    "null value",
			line:    `{"error":null,"status":"ok"}`,
			wantErr: false,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["error"] == nil && e.Fields["status"] == "ok"
			},
		},
		{
			name:    "invalid JSON",
			line:    `{invalid}`,
			wantErr: true,
		},
		{
			name:    "empty object",
			line:    `{}`,
			wantErr: false,
			checkFn: func(e *parser.LogEntry) bool {
				return len(e.Fields) == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := p.Parse(tt.line, 1)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFn != nil && !tt.checkFn(entry) {
				t.Errorf("Parse() fields check failed, got fields: %v", entry.Fields)
			}
			if entry != nil {
				parser.ReleaseEntry(entry)
			}
		})
	}
}

func TestJSONParser_CanParse(t *testing.T) {
	p := parser.NewJSONParser()

	tests := []struct {
		line string
		want bool
	}{
		{`{"level":"error"}`, true},
		{`  {"level":"error"}`, true},
		{`level=error`, false},
		{`plain text`, false},
		{``, false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := p.CanParse(tt.line); got != tt.want {
				t.Errorf("CanParse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeyValueParser_Parse(t *testing.T) {
	p := parser.NewKeyValueParser()

	tests := []struct {
		name    string
		line    string
		checkFn func(*parser.LogEntry) bool
	}{
		{
			name: "simple key=value",
			line: `level=error status=500`,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["level"] == "error" && e.Fields["status"] == int64(500)
			},
		},
		{
			name: "quoted values",
			line: `level=error msg="something happened"`,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["msg"] == "something happened"
			},
		},
		{
			name: "quoted values with spaces",
			line: `level=error msg="multi word message"`,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["msg"] == "multi word message"
			},
		},
		{
			name: "boolean values",
			line: `enabled=true disabled=false`,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["enabled"] == true && e.Fields["disabled"] == false
			},
		},
		{
			name: "float values",
			line: `memory=85.5 cpu=0.75`,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["memory"] == 85.5 && e.Fields["cpu"] == 0.75
			},
		},
		{
			name: "negative numbers",
			line: `offset=-10 temp=-5.5`,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["offset"] == int64(-10) && e.Fields["temp"] == -5.5
			},
		},
		{
			name: "dotted keys",
			line: `user.id=123 user.name=john`,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["user.id"] == int64(123) && e.Fields["user.name"] == "john"
			},
		},
		{
			name: "empty value",
			line: `level= status=200`,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["level"] == "" && e.Fields["status"] == int64(200)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := p.Parse(tt.line, 1)
			if err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
				return
			}
			if tt.checkFn != nil && !tt.checkFn(entry) {
				t.Errorf("Parse() fields check failed, got fields: %v", entry.Fields)
			}
			parser.ReleaseEntry(entry)
		})
	}
}

func TestAutoParser_Parse(t *testing.T) {
	p := parser.NewAutoParser()

	tests := []struct {
		name    string
		line    string
		checkFn func(*parser.LogEntry) bool
	}{
		{
			name: "detects JSON",
			line: `{"level":"error"}`,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["level"] == "error"
			},
		},
		{
			name: "detects key=value",
			line: `level=error status=500`,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["level"] == "error"
			},
		},
		{
			name: "handles empty line",
			line: ``,
			checkFn: func(e *parser.LogEntry) bool {
				return len(e.Fields) == 0
			},
		},
		{
			name: "handles whitespace only",
			line: `   `,
			checkFn: func(e *parser.LogEntry) bool {
				return len(e.Fields) == 0
			},
		},
		{
			name: "handles plain text",
			line: `This is a plain log message`,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Raw == "This is a plain log message"
			},
		},
		{
			name: "handles mixed format",
			line: `level=error status=500`,
			checkFn: func(e *parser.LogEntry) bool {
				return e.Fields["level"] == "error" && e.Fields["status"] == int64(500)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := p.Parse(tt.line, 1)
			if err != nil {
				t.Errorf("Parse() unexpected error = %v", err)
				return
			}
			if tt.checkFn != nil && !tt.checkFn(entry) {
				t.Errorf("Parse() fields check failed, got fields: %v", entry.Fields)
			}
			parser.ReleaseEntry(entry)
		})
	}
}

func TestLogEntry_Pool(t *testing.T) {
	// Test that pooling works correctly
	entry1 := parser.AcquireEntry()
	entry1.Raw = "test1"
	entry1.Fields["key"] = "value"
	entry1.LineNum = 1

	parser.ReleaseEntry(entry1)

	entry2 := parser.AcquireEntry()
	// Entry should be cleared
	if entry2.Raw != "" {
		t.Error("Expected Raw to be cleared")
	}
	if len(entry2.Fields) != 0 {
		t.Error("Expected Fields to be cleared")
	}
	if entry2.LineNum != 0 {
		t.Error("Expected LineNum to be cleared")
	}

	parser.ReleaseEntry(entry2)
}

// Benchmarks

func BenchmarkJSONParser(b *testing.B) {
	p := parser.NewJSONParser()
	line := `{"timestamp":"2024-01-15T10:00:02Z","level":"error","message":"Connection timeout","status":500,"user":{"id":456,"name":"jane"}}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry, _ := p.Parse(line, i)
		parser.ReleaseEntry(entry)
	}
}

func BenchmarkJSONParser_Nested(b *testing.B) {
	p := parser.NewJSONParser()
	line := `{"user":{"profile":{"settings":{"theme":"dark","notifications":{"email":true,"push":false}}}}}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry, _ := p.Parse(line, i)
		parser.ReleaseEntry(entry)
	}
}

func BenchmarkKeyValueParser(b *testing.B) {
	p := parser.NewKeyValueParser()
	line := `timestamp=2024-01-15T10:00:02Z level=error message="Connection timeout" status=500 user.id=456`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry, _ := p.Parse(line, i)
		parser.ReleaseEntry(entry)
	}
}

func BenchmarkAutoParser_JSON(b *testing.B) {
	p := parser.NewAutoParser()
	line := `{"level":"error","status":500}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry, _ := p.Parse(line, i)
		parser.ReleaseEntry(entry)
	}
}

func BenchmarkAutoParser_KeyValue(b *testing.B) {
	p := parser.NewAutoParser()
	line := `level=error status=500`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry, _ := p.Parse(line, i)
		parser.ReleaseEntry(entry)
	}
}

