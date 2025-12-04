# flog 🔍

**Fast Log Filter** - A blazingly fast CLI tool for filtering structured logs by multiple fields.

## Features

- 🔍 **Multi-field filtering** - Chain filters with AND/OR logic
- ⚡ **Instant results** - Streaming + parallel processing for large files
- 🎯 **Schema-agnostic** - Works with any JSON log structure
- 🛠️ **Unix-friendly** - Supports pipes and stdin

## Installation

```bash
go install github.com/ishk9/flog/cmd/flog@latest
```

Or build from source:

```bash
git clone https://github.com/ishk9/flog
cd flog
go build -o flog ./cmd/flog
```

## Usage

```bash
flog [OPTIONS] <FILE>...

Options:
  -f, --filter <QUERY>   Filter expression
  -o, --output <FORMAT>  Output format: raw|pretty|json
  -c, --count            Print match count only
  -n, --limit <N>        Limit to first N matches
  -h, --help             Show help
```

## Filter Syntax

```bash
# Basic equality (AND by default)
flog -f "level:error,status:500" app.log

# Nested fields
flog -f "user.profile.role:admin" events.log

# Comparison operators
flog -f "status>=400,status<500" access.log

# OR conditions
flog -f "level:error|level:warn" app.log

# Regex matching
flog -f "message~=timeout.*retry" app.log

# Field existence
flog -f "error?" app.log
```

## Examples

```bash
# Find all errors for a specific user
flog -f "level:error,user.id:12345" app.log

# Count 5xx errors
flog -f "status>=500" --count access.log

# Pretty print with selected fields
flog -f "level:error" -o pretty app.log

# Chain with other tools
cat app.log | flog -f "level:error" - | jq .message
```

## License

MIT

