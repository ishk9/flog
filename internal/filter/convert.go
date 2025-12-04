package filter

import (
	"fmt"
	"strconv"
	"strings"
)

// toString converts any value to a string representation.
func toString(v any) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		// Use %g for clean output (no trailing zeros)
		return strconv.FormatFloat(val, 'g', -1, 64)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// toFloat64 attempts to convert a value to float64.
// Returns the value and true if successful, 0 and false otherwise.
func toFloat64(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}

	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case int16:
		return float64(val), true
	case int8:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint64:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint16:
		return float64(val), true
	case uint8:
		return float64(val), true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// equalFold compares two strings case-insensitively.
func equalFold(a, b string) bool {
	return strings.EqualFold(a, b)
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// containsFold checks if s contains substr case-insensitively.
func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// ParseValue parses a string value into an appropriate type.
func ParseValue(s string) any {
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
	case "null", "nil":
		return nil
	}

	// Return as string
	return s
}

