// Package filter provides log filtering and matching functionality.
package filter

import "github.com/ishk9/flog/internal/parser"

// Operator represents the type of comparison for a filter condition.
type Operator int

const (
	OpEq       Operator = iota // Equal: field:value or field=value
	OpNe                       // Not equal: field!=value
	OpGt                       // Greater than: field>value
	OpLt                       // Less than: field<value
	OpGte                      // Greater than or equal: field>=value
	OpLte                      // Less than or equal: field<=value
	OpRegex                    // Regex match: field~=pattern
	OpContains                 // Contains substring: field*=substring
	OpExists                   // Field exists: field?
)

// Logic represents how conditions are combined.
type Logic int

const (
	LogicAnd Logic = iota // All conditions must match
	LogicOr               // Any condition can match
)

// Condition represents a single filter condition.
type Condition struct {
	Field    string   // Field path (e.g., "user.id", "level")
	Operator Operator // Comparison operator
	Value    any      // Target value to match against
}

// FilterChain represents a combination of conditions with logic.
type FilterChain struct {
	Conditions []Condition
	Logic      Logic
	SubChains  []*FilterChain // For nested AND/OR grouping
}

// Matcher evaluates filter conditions against log entries.
type Matcher interface {
	// Match checks if a log entry satisfies the filter chain.
	Match(entry *parser.LogEntry, chain *FilterChain) bool
}

