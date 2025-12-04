package parser

import (
	"strconv"
	"strings"
	"unicode"
)

// KeyValueParser parses key=value formatted log lines.
// Supports formats like: level=error msg="something happened" user.id=123
type KeyValueParser struct{}

// NewKeyValueParser creates a new key-value parser instance.
func NewKeyValueParser() *KeyValueParser {
	return &KeyValueParser{}
}

// CanParse checks if the line appears to be key=value format.
func (p *KeyValueParser) CanParse(line string) bool {
	// Look for at least one key=value pattern
	return strings.Contains(line, "=") && !strings.HasPrefix(strings.TrimSpace(line), "{")
}

// Parse converts a key=value log line into a LogEntry.
func (p *KeyValueParser) Parse(line string, lineNum int) (*LogEntry, error) {
	entry := AcquireEntry()
	entry.Raw = line
	entry.LineNum = lineNum

	p.parseKeyValues(line, entry.Fields)

	return entry, nil
}

// parseKeyValues extracts key=value pairs from a line.
func (p *KeyValueParser) parseKeyValues(line string, fields map[string]any) {
	i := 0
	n := len(line)

	for i < n {
		// Skip whitespace
		for i < n && unicode.IsSpace(rune(line[i])) {
			i++
		}
		if i >= n {
			break
		}

		// Find key (until '=' or whitespace)
		keyStart := i
		for i < n && line[i] != '=' && !unicode.IsSpace(rune(line[i])) {
			i++
		}
		if i >= n || line[i] != '=' {
			// Not a key=value pair, skip to next whitespace
			for i < n && !unicode.IsSpace(rune(line[i])) {
				i++
			}
			continue
		}

		key := line[keyStart:i]
		i++ // Skip '='

		if i >= n {
			fields[key] = ""
			break
		}

		// Parse value
		var value string
		if line[i] == '"' {
			// Quoted value
			i++ // Skip opening quote
			valueStart := i
			for i < n && line[i] != '"' {
				if line[i] == '\\' && i+1 < n {
					i++ // Skip escaped char
				}
				i++
			}
			value = line[valueStart:i]
			if i < n {
				i++ // Skip closing quote
			}
		} else {
			// Unquoted value
			valueStart := i
			for i < n && !unicode.IsSpace(rune(line[i])) {
				i++
			}
			value = line[valueStart:i]
		}

		// Try to infer type
		fields[key] = inferType(value)
	}
}

// inferType attempts to convert a string value to its appropriate type.
func inferType(s string) any {
	// Try integer
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}

	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	// Try boolean
	switch strings.ToLower(s) {
	case "true":
		return true
	case "false":
		return false
	}

	// Return as string
	return s
}

