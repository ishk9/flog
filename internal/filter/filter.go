// Package filter provides log filtering and matching functionality.
package filter

import (
	"regexp"
	"sync"

	"github.com/ishk9/flog/internal/parser"
)

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

// String returns the string representation of an operator.
func (o Operator) String() string {
	switch o {
	case OpEq:
		return "="
	case OpNe:
		return "!="
	case OpGt:
		return ">"
	case OpLt:
		return "<"
	case OpGte:
		return ">="
	case OpLte:
		return "<="
	case OpRegex:
		return "~="
	case OpContains:
		return "*="
	case OpExists:
		return "?"
	default:
		return "?"
	}
}

// Logic represents how conditions are combined.
type Logic int

const (
	LogicAnd Logic = iota // All conditions must match
	LogicOr               // Any condition can match
)

// Condition represents a single filter condition.
type Condition struct {
	Field       string         // Field path (e.g., "user.id", "level")
	Operator    Operator       // Comparison operator
	Value       any            // Target value to match against
	compiled    *regexp.Regexp // Cached compiled regex for OpRegex
	IgnoreCase  bool           // Case-insensitive matching
}

// FilterChain represents a combination of conditions with logic.
type FilterChain struct {
	Conditions []Condition
	Logic      Logic
	SubChains  []*FilterChain // For nested AND/OR grouping
}

// NewCondition creates a new filter condition.
func NewCondition(field string, op Operator, value any) Condition {
	return Condition{
		Field:    field,
		Operator: op,
		Value:    value,
	}
}

// NewFilterChain creates a new filter chain with AND logic.
func NewFilterChain(logic Logic, conditions ...Condition) *FilterChain {
	return &FilterChain{
		Conditions: conditions,
		Logic:      logic,
	}
}

// Add appends a condition to the filter chain.
func (fc *FilterChain) Add(c Condition) *FilterChain {
	fc.Conditions = append(fc.Conditions, c)
	return fc
}

// AddSubChain adds a nested filter chain.
func (fc *FilterChain) AddSubChain(sub *FilterChain) *FilterChain {
	fc.SubChains = append(fc.SubChains, sub)
	return fc
}

// Matcher evaluates filter conditions against log entries.
type Matcher struct {
	regexCache sync.Map // Cache for compiled regex patterns
	ignoreCase bool
}

// NewMatcher creates a new matcher instance.
func NewMatcher(ignoreCase bool) *Matcher {
	return &Matcher{
		ignoreCase: ignoreCase,
	}
}

// Match checks if a log entry satisfies the filter chain.
func (m *Matcher) Match(entry *parser.LogEntry, chain *FilterChain) bool {
	if chain == nil || (len(chain.Conditions) == 0 && len(chain.SubChains) == 0) {
		return true
	}

	// Evaluate main conditions
	conditionResult := m.evaluateConditions(entry, chain.Conditions, chain.Logic)

	// If no sub-chains, return condition result
	if len(chain.SubChains) == 0 {
		return conditionResult
	}

	// Evaluate sub-chains with the same logic
	for _, sub := range chain.SubChains {
		subResult := m.Match(entry, sub)

		switch chain.Logic {
		case LogicAnd:
			if !subResult {
				return false
			}
		case LogicOr:
			if subResult {
				return true
			}
		}
	}

	// Combine results based on logic
	switch chain.Logic {
	case LogicAnd:
		return conditionResult
	case LogicOr:
		return conditionResult
	}

	return conditionResult
}

// evaluateConditions evaluates a slice of conditions with the given logic.
func (m *Matcher) evaluateConditions(entry *parser.LogEntry, conditions []Condition, logic Logic) bool {
	if len(conditions) == 0 {
		return true
	}

	for _, cond := range conditions {
		result := m.evaluateCondition(entry, &cond)

		switch logic {
		case LogicAnd:
			if !result {
				return false // Short-circuit: one false makes AND false
			}
		case LogicOr:
			if result {
				return true // Short-circuit: one true makes OR true
			}
		}
	}

	// If we get here:
	// - For AND: all conditions were true
	// - For OR: no condition was true
	return logic == LogicAnd
}

// evaluateCondition evaluates a single condition against an entry.
func (m *Matcher) evaluateCondition(entry *parser.LogEntry, cond *Condition) bool {
	// Handle existence check
	if cond.Operator == OpExists {
		_, exists := entry.Fields[cond.Field]
		return exists
	}

	// Get field value
	fieldValue, exists := entry.Fields[cond.Field]
	if !exists {
		return false
	}

	return m.compareValues(fieldValue, cond)
}

// compareValues compares a field value against a condition.
func (m *Matcher) compareValues(fieldValue any, cond *Condition) bool {
	switch cond.Operator {
	case OpEq:
		return m.equal(fieldValue, cond.Value, cond.IgnoreCase || m.ignoreCase)
	case OpNe:
		return !m.equal(fieldValue, cond.Value, cond.IgnoreCase || m.ignoreCase)
	case OpGt:
		return m.compare(fieldValue, cond.Value) > 0
	case OpLt:
		return m.compare(fieldValue, cond.Value) < 0
	case OpGte:
		return m.compare(fieldValue, cond.Value) >= 0
	case OpLte:
		return m.compare(fieldValue, cond.Value) <= 0
	case OpRegex:
		return m.matchRegex(fieldValue, cond)
	case OpContains:
		return m.contains(fieldValue, cond.Value, cond.IgnoreCase || m.ignoreCase)
	default:
		return false
	}
}

// equal checks if two values are equal.
func (m *Matcher) equal(a, b any, ignoreCase bool) bool {
	aStr := toString(a)
	bStr := toString(b)

	if ignoreCase {
		return equalFold(aStr, bStr)
	}
	return aStr == bStr
}

// compare compares two values, returning -1, 0, or 1.
func (m *Matcher) compare(a, b any) int {
	// Try numeric comparison first
	aNum, aIsNum := toFloat64(a)
	bNum, bIsNum := toFloat64(b)

	if aIsNum && bIsNum {
		if aNum < bNum {
			return -1
		} else if aNum > bNum {
			return 1
		}
		return 0
	}

	// Fall back to string comparison
	aStr := toString(a)
	bStr := toString(b)

	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

// matchRegex matches a field value against a regex pattern.
func (m *Matcher) matchRegex(fieldValue any, cond *Condition) bool {
	// Get or compile regex
	pattern := toString(cond.Value)

	var re *regexp.Regexp
	if cached, ok := m.regexCache.Load(pattern); ok {
		re = cached.(*regexp.Regexp)
	} else {
		var err error
		if cond.IgnoreCase || m.ignoreCase {
			re, err = regexp.Compile("(?i)" + pattern)
		} else {
			re, err = regexp.Compile(pattern)
		}
		if err != nil {
			return false
		}
		m.regexCache.Store(pattern, re)
	}

	return re.MatchString(toString(fieldValue))
}

// contains checks if a string contains a substring.
func (m *Matcher) contains(fieldValue, searchValue any, ignoreCase bool) bool {
	fieldStr := toString(fieldValue)
	searchStr := toString(searchValue)

	if ignoreCase {
		return containsFold(fieldStr, searchStr)
	}
	return containsString(fieldStr, searchStr)
}
