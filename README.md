# flog 🔍

**Fast Log Filter** - A blazingly fast CLI tool for filtering structured logs by multiple fields.

## Features

- 🔍 **Multi-field filtering** - Chain filters with AND/OR logic
- ⚡ **Instant results** - Streaming + parallel processing for large files
- 🎯 **Schema-agnostic** - Works with any JSON log structure
- 🛠️ **Unix-friendly** - Supports pipes and stdin
- 🎨 **Pretty output** - Colorized JSON, field selection, count mode

## Installation

### Quick Install (Recommended)

```bash
curl -sSL https://raw.githubusercontent.com/ishk9/flog/main/install.sh | bash
```

### Homebrew (macOS/Linux)

```bash
brew tap ishk9/tap
brew install flog
```

### Go Install

```bash
go install github.com/ishk9/flog/cmd/flog@latest
```

### Download Binary

Download pre-built binaries from [GitHub Releases](https://github.com/ishk9/flog/releases).

### Build from Source

```bash
git clone https://github.com/ishk9/flog
cd flog
go build -o flog ./cmd/flog
```

## Quick Start

```bash
# Find all error logs
flog -f "level:error" app.log

# Find errors with status >= 500
flog -f "level:error,status>=500" app.log

# Find errors OR warnings
flog -f "level:error|level:warn" app.log

# Filter nested fields
flog -f "user.profile.role:admin" events.log

# Count matches
flog -f "status>=400" --count access.log

# Pretty print results
flog -f "level:error" -o pretty app.log
```

## Filter Syntax

### Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `:` or `=` | Equals | `level:error` |
| `!=` | Not equals | `level!=debug` |
| `>` | Greater than | `status>400` |
| `<` | Less than | `status<500` |
| `>=` | Greater than or equal | `status>=400` |
| `<=` | Less than or equal | `status<=599` |
| `~=` | Regex match | `message~=timeout.*retry` |
| `*=` | Contains substring | `message*=timeout` |
| `?` | Field exists | `error?` |

### Combining Filters

| Syntax | Meaning | Example |
|--------|---------|---------|
| `,` | AND | `level:error,status:500` |
| `\|` | OR | `level:error\|level:warn` |

### Nested Fields

Access nested JSON fields with dot notation:

```bash
# JSON: {"user": {"profile": {"role": "admin"}}}
flog -f "user.profile.role:admin" events.log
```

## Usage

```
flog [OPTIONS] <FILE>...

Arguments:
  <FILE>...              Log file(s) to filter (use - for stdin)

Options:
  -f, --filter <QUERY>   Filter expression (required)
  -o, --output <FORMAT>  Output format: raw|pretty|json|fields [default: raw]
  -F, --fields <FIELDS>  Comma-separated fields to output
  -c, --count            Print match count only
  -n, --limit <N>        Limit output to first N matches
  -i, --ignore-case      Case-insensitive matching
  -v, --invert           Invert match (print non-matching lines)
  -j, --jobs <N>         Number of parallel workers [default: CPU count]
      --stats            Print filter statistics
      --no-color         Disable colored output
  -h, --help             Print help
  -V, --version          Print version
```

## Examples

### Basic Filtering

```bash
# Find all errors
flog -f "level:error" app.log

# Find specific status codes
flog -f "status:404" access.log

# Multiple conditions (AND)
flog -f "level:error,user.id:123" app.log
```

### Comparison Operators

```bash
# Find 5xx errors
flog -f "status>=500" access.log

# Find client errors (4xx)
flog -f "status>=400,status<500" access.log
```

### OR Conditions

```bash
# Errors or warnings
flog -f "level:error|level:warn" app.log

# Multiple users
flog -f "user.id:123|user.id:456" events.log
```

### Regex Matching

```bash
# Messages containing "timeout" followed by "retry"
flog -f "message~=timeout.*retry" app.log

# Path starting with /api/
flog -f "path~=^/api/" access.log
```

### Output Formatting

```bash
# Pretty print with colors
flog -f "level:error" -o pretty app.log

# Extract specific fields
flog -f "level:error" -F "timestamp,message,status" app.log

# Count matches only
flog -f "level:error" --count app.log

# Show statistics
flog -f "level:error" --stats app.log
```

### Piping & Chaining

```bash
# Read from stdin
cat app.log | flog -f "level:error" -

# Chain filters
flog -f "status>=400" access.log | flog -f "method:POST" -

# Combine with other tools
flog -f "level:error" app.log | jq .message
```

## Supported Log Formats

### JSON Logs

```json
{"timestamp":"2024-01-15T10:00:00Z","level":"error","message":"Connection failed","user":{"id":123}}
```

### Key-Value Logs

```
timestamp=2024-01-15T10:00:00Z level=error message="Connection failed" user.id=123
```

## Performance

flog is optimized for speed:

- **Streaming:** Processes files line-by-line without loading into memory
- **Object pooling:** Reuses memory allocations
- **Regex caching:** Compiles patterns once
- **Parallel processing:** Uses all CPU cores by default

## Documentation

- [Low-Level Design](docs/LLD.md) - Architecture and implementation details

## License

MIT
