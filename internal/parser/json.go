package parser

import (
	"encoding/json"
	"strings"
)

// JSONParser parses JSON-formatted log lines.
type JSONParser struct{}

// NewJSONParser creates a new JSON parser instance.
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// CanParse checks if the line appears to be valid JSON.
func (p *JSONParser) CanParse(line string) bool {
	line = strings.TrimSpace(line)
	return len(line) > 0 && line[0] == '{'
}

// Parse converts a JSON log line into a LogEntry with flattened fields.
func (p *JSONParser) Parse(line string, lineNum int) (*LogEntry, error) {
	entry := AcquireEntry()
	entry.Raw = line
	entry.LineNum = lineNum

	var data map[string]any
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		ReleaseEntry(entry)
		return nil, err
	}

	// Flatten nested structure
	flattenMap("", data, entry.Fields)

	return entry, nil
}

// flattenMap recursively flattens a nested map into dot-notation keys.
// Example: {"user": {"id": 123}} -> {"user.id": 123}
func flattenMap(prefix string, data map[string]any, result map[string]any) {
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]any:
			// Recurse into nested objects
			flattenMap(fullKey, v, result)
			// Also store the nested object itself for existence checks
			result[fullKey] = v
		case []any:
			// Store arrays as-is, also flatten array elements if they're maps
			result[fullKey] = v
			for i, item := range v {
				if m, ok := item.(map[string]any); ok {
					arrayPrefix := fullKey + "[" + itoa(i) + "]"
					flattenMap(arrayPrefix, m, result)
				}
			}
		default:
			result[fullKey] = v
		}
	}
}

// itoa converts int to string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}

