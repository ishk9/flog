package tests

import (
	"testing"

	"github.com/ishk9/flog/internal/filter"
	"github.com/ishk9/flog/internal/parser"
)

func TestMatcher_Match(t *testing.T) {
	matcher := filter.NewMatcher(false)

	entry := parser.NewLogEntry(`{"level":"error","status":500}`, 1)
	entry.Fields["level"] = "error"
	entry.Fields["status"] = float64(500)
	entry.Fields["user.id"] = float64(123)
	entry.Fields["message"] = "Connection timeout"

	tests := []struct {
		name      string
		chain     *filter.FilterChain
		wantMatch bool
	}{
		{
			name:      "simple equality",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("level", filter.OpEq, "error")),
			wantMatch: true,
		},
		{
			name:      "simple equality no match",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("level", filter.OpEq, "info")),
			wantMatch: false,
		},
		{
			name:      "not equal",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("level", filter.OpNe, "info")),
			wantMatch: true,
		},
		{
			name:      "not equal - should not match",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("level", filter.OpNe, "error")),
			wantMatch: false,
		},
		{
			name:      "greater than",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("status", filter.OpGt, float64(400))),
			wantMatch: true,
		},
		{
			name:      "greater than - boundary",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("status", filter.OpGt, float64(500))),
			wantMatch: false,
		},
		{
			name:      "greater than or equal",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("status", filter.OpGte, float64(500))),
			wantMatch: true,
		},
		{
			name:      "less than",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("status", filter.OpLt, float64(600))),
			wantMatch: true,
		},
		{
			name:      "less than - boundary",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("status", filter.OpLt, float64(500))),
			wantMatch: false,
		},
		{
			name:      "less than or equal",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("status", filter.OpLte, float64(500))),
			wantMatch: true,
		},
		{
			name:      "AND logic - all match",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("level", filter.OpEq, "error"), filter.NewCondition("status", filter.OpGte, float64(500))),
			wantMatch: true,
		},
		{
			name:      "AND logic - one fails",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("level", filter.OpEq, "error"), filter.NewCondition("status", filter.OpLt, float64(400))),
			wantMatch: false,
		},
		{
			name:      "OR logic - one matches",
			chain:     filter.NewFilterChain(filter.LogicOr, filter.NewCondition("level", filter.OpEq, "info"), filter.NewCondition("level", filter.OpEq, "error")),
			wantMatch: true,
		},
		{
			name:      "OR logic - none match",
			chain:     filter.NewFilterChain(filter.LogicOr, filter.NewCondition("level", filter.OpEq, "info"), filter.NewCondition("level", filter.OpEq, "warn")),
			wantMatch: false,
		},
		{
			name:      "nested field",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("user.id", filter.OpEq, float64(123))),
			wantMatch: true,
		},
		{
			name:      "regex match",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("message", filter.OpRegex, "timeout")),
			wantMatch: true,
		},
		{
			name:      "regex match - case sensitive",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("message", filter.OpRegex, "Timeout")),
			wantMatch: false,
		},
		{
			name:      "regex match - pattern",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("message", filter.OpRegex, "^Connection.*$")),
			wantMatch: true,
		},
		{
			name:      "contains",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("message", filter.OpContains, "timeout")),
			wantMatch: true,
		},
		{
			name:      "contains - case sensitive",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("message", filter.OpContains, "Timeout")),
			wantMatch: false,
		},
		{
			name:      "field exists",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("status", filter.OpExists, nil)),
			wantMatch: true,
		},
		{
			name:      "field not exists",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("nonexistent", filter.OpExists, nil)),
			wantMatch: false,
		},
		{
			name:      "empty chain matches all",
			chain:     filter.NewFilterChain(filter.LogicAnd),
			wantMatch: true,
		},
		{
			name:      "nil chain matches all",
			chain:     nil,
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matcher.Match(entry, tt.chain)
			if got != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestMatcher_CaseInsensitive(t *testing.T) {
	matcher := filter.NewMatcher(true)

	entry := parser.NewLogEntry(`{"level":"ERROR"}`, 1)
	entry.Fields["level"] = "ERROR"
	entry.Fields["message"] = "Connection TIMEOUT"

	tests := []struct {
		name      string
		chain     *filter.FilterChain
		wantMatch bool
	}{
		{
			name:      "equality case insensitive",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("level", filter.OpEq, "error")),
			wantMatch: true,
		},
		{
			name:      "contains case insensitive",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("message", filter.OpContains, "timeout")),
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matcher.Match(entry, tt.chain)
			if got != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestMatcher_TypeCoercion(t *testing.T) {
	matcher := filter.NewMatcher(false)

	entry := parser.NewLogEntry(`{"status":500}`, 1)
	entry.Fields["status"] = float64(500)
	entry.Fields["port"] = "8080"

	tests := []struct {
		name      string
		chain     *filter.FilterChain
		wantMatch bool
	}{
		{
			name:      "number equality with string value",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("status", filter.OpEq, "500")),
			wantMatch: true,
		},
		{
			name:      "string equality with number comparison",
			chain:     filter.NewFilterChain(filter.LogicAnd, filter.NewCondition("port", filter.OpGt, "8000")),
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matcher.Match(entry, tt.chain)
			if got != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestQueryParser_Parse(t *testing.T) {
	qp := filter.NewQueryParser()

	tests := []struct {
		name    string
		query   string
		wantErr bool
		check   func(*filter.FilterChain) bool
	}{
		{
			name:    "simple equality with colon",
			query:   "level:error",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return len(fc.Conditions) == 1 && fc.Conditions[0].Field == "level"
			},
		},
		{
			name:    "simple equality with equals",
			query:   "level=error",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return len(fc.Conditions) == 1 && fc.Conditions[0].Operator == filter.OpEq
			},
		},
		{
			name:    "not equal",
			query:   "level!=error",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return len(fc.Conditions) == 1 && fc.Conditions[0].Operator == filter.OpNe
			},
		},
		{
			name:    "greater than",
			query:   "status>400",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return len(fc.Conditions) == 1 && fc.Conditions[0].Operator == filter.OpGt
			},
		},
		{
			name:    "greater than or equal",
			query:   "status>=400",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return len(fc.Conditions) == 1 && fc.Conditions[0].Operator == filter.OpGte
			},
		},
		{
			name:    "less than",
			query:   "status<500",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return len(fc.Conditions) == 1 && fc.Conditions[0].Operator == filter.OpLt
			},
		},
		{
			name:    "less than or equal",
			query:   "status<=500",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return len(fc.Conditions) == 1 && fc.Conditions[0].Operator == filter.OpLte
			},
		},
		{
			name:    "AND with comma",
			query:   "level:error,status:500",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return len(fc.Conditions) == 2 && fc.Logic == filter.LogicAnd
			},
		},
		{
			name:    "OR with pipe",
			query:   "level:error|level:warn",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return len(fc.SubChains) == 1 && fc.SubChains[0].Logic == filter.LogicOr
			},
		},
		{
			name:    "regex operator",
			query:   "message~=timeout.*retry",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return len(fc.Conditions) == 1 && fc.Conditions[0].Operator == filter.OpRegex
			},
		},
		{
			name:    "contains operator",
			query:   "message*=timeout",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return len(fc.Conditions) == 1 && fc.Conditions[0].Operator == filter.OpContains
			},
		},
		{
			name:    "exists operator",
			query:   "error?",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return len(fc.Conditions) == 1 && fc.Conditions[0].Operator == filter.OpExists
			},
		},
		{
			name:    "nested field",
			query:   "user.profile.role:admin",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return len(fc.Conditions) == 1 && fc.Conditions[0].Field == "user.profile.role"
			},
		},
		{
			name:    "quoted value with double quotes",
			query:   `message:"hello world"`,
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return fc.Conditions[0].Value == "hello world"
			},
		},
		{
			name:    "quoted value with single quotes",
			query:   `message:'hello world'`,
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return fc.Conditions[0].Value == "hello world"
			},
		},
		{
			name:    "numeric value",
			query:   "status:500",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return fc.Conditions[0].Value == int64(500)
			},
		},
		{
			name:    "boolean true value",
			query:   "active:true",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return fc.Conditions[0].Value == true
			},
		},
		{
			name:    "boolean false value",
			query:   "active:false",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				return fc.Conditions[0].Value == false
			},
		},
		{
			name:    "complex AND/OR",
			query:   "level:error|level:warn,status>=400",
			wantErr: false,
			check: func(fc *filter.FilterChain) bool {
				// Should have OR subchain and status condition
				return len(fc.SubChains) == 1 && len(fc.Conditions) == 1
			},
		},
		{
			name:    "empty query",
			query:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			query:   "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, err := qp.Parse(tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(chain) {
				t.Errorf("Parse() check failed for query: %s, got chain: %+v", tt.query, chain)
			}
		})
	}
}

func TestFilterChain_Methods(t *testing.T) {
	chain := filter.NewFilterChain(filter.LogicAnd)

	// Test Add
	chain.Add(filter.NewCondition("level", filter.OpEq, "error"))
	if len(chain.Conditions) != 1 {
		t.Error("Add() should add condition")
	}

	// Test AddSubChain
	subChain := filter.NewFilterChain(filter.LogicOr)
	chain.AddSubChain(subChain)
	if len(chain.SubChains) != 1 {
		t.Error("AddSubChain() should add sub-chain")
	}
}

// Benchmarks

func BenchmarkMatcher(b *testing.B) {
	matcher := filter.NewMatcher(false)
	entry := parser.NewLogEntry(`{"level":"error","status":500}`, 1)
	entry.Fields["level"] = "error"
	entry.Fields["status"] = float64(500)

	chain := filter.NewFilterChain(filter.LogicAnd,
		filter.NewCondition("level", filter.OpEq, "error"),
		filter.NewCondition("status", filter.OpGte, float64(400)),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Match(entry, chain)
	}
}

func BenchmarkMatcher_Regex(b *testing.B) {
	matcher := filter.NewMatcher(false)
	entry := parser.NewLogEntry(`{"message":"Connection timeout after retry"}`, 1)
	entry.Fields["message"] = "Connection timeout after retry"

	chain := filter.NewFilterChain(filter.LogicAnd,
		filter.NewCondition("message", filter.OpRegex, "timeout.*retry"),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Match(entry, chain)
	}
}

func BenchmarkQueryParser(b *testing.B) {
	qp := filter.NewQueryParser()
	query := "level:error,status>=400,user.id:123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qp.Parse(query)
	}
}

func BenchmarkQueryParser_Complex(b *testing.B) {
	qp := filter.NewQueryParser()
	query := "level:error|level:warn,status>=400,user.profile.role:admin"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qp.Parse(query)
	}
}

