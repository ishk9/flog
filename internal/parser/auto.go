package parser

import "strings"

// AutoParser automatically detects and uses the appropriate parser.
type AutoParser struct {
	jsonParser     *JSONParser
	keyValueParser *KeyValueParser
}

// NewAutoParser creates a new auto-detecting parser.
func NewAutoParser() *AutoParser {
	return &AutoParser{
		jsonParser:     NewJSONParser(),
		keyValueParser: NewKeyValueParser(),
	}
}

// CanParse always returns true as AutoParser handles all formats.
func (p *AutoParser) CanParse(line string) bool {
	return true
}

// Parse detects the format and uses the appropriate parser.
func (p *AutoParser) Parse(line string, lineNum int) (*LogEntry, error) {
	line = strings.TrimSpace(line)

	if len(line) == 0 {
		// Empty line - return entry with just the raw line
		entry := AcquireEntry()
		entry.Raw = line
		entry.LineNum = lineNum
		return entry, nil
	}

	// Try JSON first (most structured)
	if p.jsonParser.CanParse(line) {
		if entry, err := p.jsonParser.Parse(line, lineNum); err == nil {
			return entry, nil
		}
		// Fall through to key-value if JSON parsing fails
	}

	// Try key-value format
	if p.keyValueParser.CanParse(line) {
		return p.keyValueParser.Parse(line, lineNum)
	}

	// Fallback: return entry with just raw line (no fields)
	entry := AcquireEntry()
	entry.Raw = line
	entry.LineNum = lineNum
	return entry, nil
}

