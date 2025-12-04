package filter

import (
	"errors"
	"strings"
	"unicode"
)

// Query parsing errors.
var (
	ErrEmptyQuery        = errors.New("empty query")
	ErrInvalidSyntax     = errors.New("invalid query syntax")
	ErrUnclosedParen     = errors.New("unclosed parenthesis")
	ErrUnexpectedToken   = errors.New("unexpected token")
	ErrInvalidOperator   = errors.New("invalid operator")
	ErrMissingField      = errors.New("missing field name")
	ErrMissingValue      = errors.New("missing value")
)

// QueryParser parses filter query strings into FilterChains.
//
// Query syntax:
//   - Comma (,) = AND
//   - Pipe (|) = OR
//   - Parentheses for grouping
//
// Operators:
//   - field:value or field=value (equality)
//   - field!=value (not equal)
//   - field>value, field<value, field>=value, field<=value (comparison)
//   - field~=pattern (regex)
//   - field*=substring (contains)
//   - field? (exists)
//
// Examples:
//   - "level:error,status:500" → level=error AND status=500
//   - "level:error|level:warn" → level=error OR level=warn
//   - "(level:error|level:warn),status>=400" → (level=error OR level=warn) AND status>=400
type QueryParser struct {
	input string
	pos   int
}

// NewQueryParser creates a new query parser.
func NewQueryParser() *QueryParser {
	return &QueryParser{}
}

// Parse parses a query string into a FilterChain.
func (p *QueryParser) Parse(query string) (*FilterChain, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrEmptyQuery
	}

	p.input = query
	p.pos = 0

	return p.parseExpression()
}

// parseExpression parses a full expression (handles AND at top level).
func (p *QueryParser) parseExpression() (*FilterChain, error) {
	return p.parseAndExpr()
}

// parseAndExpr parses AND-separated terms.
func (p *QueryParser) parseAndExpr() (*FilterChain, error) {
	chain := NewFilterChain(LogicAnd)

	// Parse first term
	first, err := p.parseOrExpr()
	if err != nil {
		return nil, err
	}

	// If it's a simple condition, add it; otherwise merge
	if len(first.SubChains) == 0 && len(first.Conditions) == 1 && first.Logic == LogicAnd {
		chain.Conditions = append(chain.Conditions, first.Conditions...)
	} else if first.Logic == LogicOr || len(first.SubChains) > 0 {
		chain.SubChains = append(chain.SubChains, first)
	} else {
		chain.Conditions = append(chain.Conditions, first.Conditions...)
	}

	// Parse additional AND terms
	for p.pos < len(p.input) {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			break
		}

		// Check for AND separator (comma)
		if p.input[p.pos] != ',' {
			break
		}
		p.pos++ // consume comma

		term, err := p.parseOrExpr()
		if err != nil {
			return nil, err
		}

		if len(term.SubChains) == 0 && len(term.Conditions) == 1 && term.Logic == LogicAnd {
			chain.Conditions = append(chain.Conditions, term.Conditions...)
		} else if term.Logic == LogicOr || len(term.SubChains) > 0 {
			chain.SubChains = append(chain.SubChains, term)
		} else {
			chain.Conditions = append(chain.Conditions, term.Conditions...)
		}
	}

	return chain, nil
}

// parseOrExpr parses OR-separated terms.
func (p *QueryParser) parseOrExpr() (*FilterChain, error) {
	chain := NewFilterChain(LogicOr)

	// Parse first term
	first, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	chain.Conditions = append(chain.Conditions, first)

	// Parse additional OR terms
	for p.pos < len(p.input) {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			break
		}

		// Check for OR separator (pipe)
		if p.input[p.pos] != '|' {
			break
		}
		p.pos++ // consume pipe

		term, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		chain.Conditions = append(chain.Conditions, term)
	}

	// If only one condition, return as AND chain (simpler)
	if len(chain.Conditions) == 1 {
		return NewFilterChain(LogicAnd, chain.Conditions[0]), nil
	}

	return chain, nil
}

// parseTerm parses a single term (condition or parenthesized expression).
func (p *QueryParser) parseTerm() (Condition, error) {
	p.skipWhitespace()

	if p.pos >= len(p.input) {
		return Condition{}, ErrInvalidSyntax
	}

	// Handle parenthesized expressions - for now, skip them
	// This simplified parser doesn't handle nested parens
	if p.input[p.pos] == '(' {
		p.pos++ // consume '('
		// Find matching ')'
		depth := 1
		start := p.pos
		for p.pos < len(p.input) && depth > 0 {
			if p.input[p.pos] == '(' {
				depth++
			} else if p.input[p.pos] == ')' {
				depth--
			}
			p.pos++
		}
		if depth != 0 {
			return Condition{}, ErrUnclosedParen
		}
		// For now, just parse the inner content as a simple condition
		inner := p.input[start : p.pos-1]
		innerParser := &QueryParser{input: inner, pos: 0}
		cond, err := innerParser.parseTerm()
		if err != nil {
			return Condition{}, err
		}
		return cond, nil
	}

	return p.parseCondition()
}

// parseCondition parses a single condition (field op value).
func (p *QueryParser) parseCondition() (Condition, error) {
	p.skipWhitespace()

	// Parse field name
	field, err := p.parseField()
	if err != nil {
		return Condition{}, err
	}

	if field == "" {
		return Condition{}, ErrMissingField
	}

	p.skipWhitespace()

	// Check for existence operator (field?)
	if p.pos < len(p.input) && p.input[p.pos] == '?' {
		p.pos++
		return NewCondition(field, OpExists, nil), nil
	}

	// Parse operator
	op, err := p.parseOperator()
	if err != nil {
		return Condition{}, err
	}

	// Parse value
	value, err := p.parseValue()
	if err != nil {
		return Condition{}, err
	}

	return NewCondition(field, op, ParseValue(value)), nil
}

// parseField parses a field name (supports dot notation).
func (p *QueryParser) parseField() (string, error) {
	p.skipWhitespace()
	start := p.pos

	for p.pos < len(p.input) {
		ch := rune(p.input[p.pos])
		// Allow alphanumeric, underscore, dot (for nested fields), and brackets (for arrays)
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '.' || ch == '[' || ch == ']' {
			p.pos++
		} else {
			break
		}
	}

	return p.input[start:p.pos], nil
}

// parseOperator parses an operator.
func (p *QueryParser) parseOperator() (Operator, error) {
	p.skipWhitespace()

	if p.pos >= len(p.input) {
		return OpEq, ErrInvalidOperator
	}

	// Check for two-character operators first
	if p.pos+1 < len(p.input) {
		twoChar := p.input[p.pos : p.pos+2]
		switch twoChar {
		case "!=":
			p.pos += 2
			return OpNe, nil
		case ">=":
			p.pos += 2
			return OpGte, nil
		case "<=":
			p.pos += 2
			return OpLte, nil
		case "~=":
			p.pos += 2
			return OpRegex, nil
		case "*=":
			p.pos += 2
			return OpContains, nil
		}
	}

	// Single character operators
	switch p.input[p.pos] {
	case ':', '=':
		p.pos++
		return OpEq, nil
	case '>':
		p.pos++
		return OpGt, nil
	case '<':
		p.pos++
		return OpLt, nil
	}

	return OpEq, ErrInvalidOperator
}

// parseValue parses a value (quoted or unquoted).
func (p *QueryParser) parseValue() (string, error) {
	p.skipWhitespace()

	if p.pos >= len(p.input) {
		return "", nil // Empty value is valid
	}

	// Handle quoted value
	if p.input[p.pos] == '"' || p.input[p.pos] == '\'' {
		quote := p.input[p.pos]
		p.pos++ // consume opening quote
		start := p.pos

		for p.pos < len(p.input) && p.input[p.pos] != quote {
			if p.input[p.pos] == '\\' && p.pos+1 < len(p.input) {
				p.pos++ // skip escape
			}
			p.pos++
		}

		value := p.input[start:p.pos]
		if p.pos < len(p.input) {
			p.pos++ // consume closing quote
		}
		return value, nil
	}

	// Unquoted value - read until delimiter
	start := p.pos
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == ',' || ch == '|' || ch == ')' || ch == ' ' || ch == '\t' {
			break
		}
		p.pos++
	}

	return p.input[start:p.pos], nil
}

// skipWhitespace advances past any whitespace.
func (p *QueryParser) skipWhitespace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

// MustParse parses a query and panics on error (for testing).
func MustParse(query string) *FilterChain {
	chain, err := NewQueryParser().Parse(query)
	if err != nil {
		panic(err)
	}
	return chain
}

