# flog - Fast Log Filter CLI Tool

## Low-Level Design Document

---

## 1. Overview

**flog** is a high-performance CLI tool for filtering structured logs by multiple fields with chainable filters. It's designed to handle large log files efficiently through streaming and parallel processing.

### Key Goals
- 🔍 Multi-field filtering with AND/OR logic
- ⚡ Instant filtering of large files (streaming, no full file load)
- 🎯 Schema-agnostic (works with any JSON structure)
- 🛠️ Unix-friendly (pipes, stdin support)

---

## 2. Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                           CLI Layer                              │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐  │
│  │   Args      │  │   Help/      │  │   Config               │  │
│  │   Parser    │  │   Examples   │  │   (future)             │  │
│  └─────────────┘  └──────────────┘  └────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Query Engine                             │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐  │
│  │   Query     │  │   Filter     │  │   Matcher              │  │
│  │   Parser    │  │   Chain      │  │   (exact/regex/range)  │  │
│  └─────────────┘  └──────────────┘  └────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Core Engine                              │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐  │
│  │   Log       │  │   Streaming  │  │   Parallel             │  │
│  │   Parser    │  │   Reader     │  │   Processor            │  │
│  └─────────────┘  └──────────────┘  └────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Output Layer                              │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐  │
│  │   Pretty    │  │   JSON       │  │   Stats/               │  │
│  │   Printer   │  │   Output     │  │   Count                │  │
│  └─────────────┘  └──────────────┘  └────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. Module Design

### 3.1 Log Parser (`internal/parser`)

**Responsibility:** Parse log lines into queryable structures

```go
// LogEntry represents a parsed log line
type LogEntry struct {
    Raw      string         // Original line
    Fields   map[string]any // Parsed fields (flattened)
    LineNum  int            // Line number in file
}

// Parser interface for different log formats
type Parser interface {
    Parse(line string, lineNum int) (*LogEntry, error)
    CanParse(line string) bool
}

// Supported parsers:
// - JSONParser     → {"level": "error", "user": {"id": 123}}
// - KeyValueParser → level=error user.id=123
// - AutoParser     → Auto-detect format per line
```

**Field Flattening:**
```
Input:  {"user": {"profile": {"name": "john"}}}
Output: map["user.profile.name"] = "john"
```

**Object Pooling:**
```go
// For high-performance scenarios, LogEntry objects are pooled
entry := parser.AcquireEntry()
defer parser.ReleaseEntry(entry)
```

### 3.2 Filter Engine (`internal/filter`)

**Responsibility:** Match log entries against filter conditions

```go
// Condition represents a single filter condition
type Condition struct {
    Field    string      // e.g., "user.id", "level"
    Operator Operator    // EQ, NE, GT, LT, REGEX, CONTAINS
    Value    any         // Target value
}

// Operator types
type Operator int
const (
    OpEq       Operator = iota  // field:value or field=value
    OpNe                        // field!=value
    OpGt                        // field>value
    OpLt                        // field<value
    OpGte                       // field>=value
    OpLte                       // field<=value
    OpRegex                     // field~=pattern
    OpContains                  // field*=substring
    OpExists                    // field?
)

// FilterChain represents AND/OR combinations
type FilterChain struct {
    Conditions []Condition
    Logic      Logic  // AND / OR
    SubChains  []*FilterChain
}

// Matcher evaluates conditions against entries
type Matcher struct {
    regexCache sync.Map  // Cached compiled regex patterns
    ignoreCase bool
}
```

### 3.3 Query DSL (`internal/filter/query.go`)

**Syntax Design:**

```bash
# Basic equality (AND by default)
flog -f "level:error,status:500" access.log

# Explicit operators
flog -f "status>=400,status<500" access.log

# OR conditions (use | separator)
flog -f "level:error|level:warn" access.log

# Nested fields
flog -f "user.profile.role:admin" access.log

# Regex matching
flog -f "message~=timeout.*retry" access.log

# Existence check
flog -f "error?" access.log  # Has 'error' field

# Negation
flog -f "level!=debug" access.log

# Contains
flog -f "message*=timeout" access.log
```

**Query Grammar (BNF):**
```
query      → andExpr
andExpr    → orExpr ("," orExpr)*
orExpr     → term ("|" term)*
term       → condition | "(" query ")"
condition  → field operator value | field "?"
field      → identifier ("." identifier)*
operator   → ":" | "=" | "!=" | ">" | "<" | ">=" | "<=" | "~=" | "*="
value      → quoted_string | unquoted_string | number | boolean
```

### 3.4 Streaming Reader (`internal/parser/reader.go`)

**Responsibility:** Read large files without memory bloat

```go
// StreamReader reads files line by line with efficient buffering
type StreamReader struct {
    BufferSize int  // Default: 64KB per line buffer
}

func (r *StreamReader) ReadLines(ctx context.Context, path string) (<-chan string, <-chan error)
func (r *StreamReader) ReadChunks(ctx context.Context, path string, chunkSize int) (<-chan []string, <-chan error)

// Supports:
// - Regular files
// - Gzip compressed files (.gz)
// - Stdin (when path is "-")
```

### 3.5 Parallel Processor (`internal/filter/parallel.go`)

**Strategy:** Fan-out/fan-in with worker pools

```go
type ParallelFilter struct {
    Workers   int          // Default: runtime.NumCPU()
    ChunkSize int          // Lines per chunk (default: 1000)
    parser    parser.Parser
    matcher   *Matcher
}

func (pf *ParallelFilter) Filter(
    ctx context.Context,
    lines <-chan string,
    chain *FilterChain,
) <-chan *parser.LogEntry
```

**Pipeline Flow:**
```
File → [Stream Reader] → [Worker Pool] → [Merger] → Output
            │                  │
            │            ┌─────┴─────┐
            │            │ Worker 1  │
            └───────────▶│ Worker 2  │───────▶ Results
                         │ Worker N  │
                         └───────────┘
```

### 3.6 Output Formatter (`internal/output`)

```go
type Formatter interface {
    Format(entry *parser.LogEntry) string
}

// Implementations:
// - RawFormatter    → Original line
// - PrettyFormatter → Colorized, indented JSON
// - JSONFormatter   → Compact JSON
// - FieldsFormatter → Only selected fields
```

---

## 4. CLI Interface

### 4.1 Command Structure

```bash
flog [OPTIONS] <FILE>...

Arguments:
  <FILE>...  Log file(s) to filter (use - for stdin)

Options:
  -f, --filter <QUERY>      Filter expression (required)
  -o, --output <FORMAT>     Output format: raw|pretty|json|fields [default: raw]
  -c, --count               Print match count only
  -n, --limit <N>           Limit output to first N matches
  -F, --fields <FIELDS>     Select specific fields to output
  -i, --ignore-case         Case-insensitive matching
  -v, --invert              Invert match (print non-matching)
  -j, --jobs <N>            Parallel workers [default: CPU count]
      --stats               Print filter statistics
      --no-color            Disable colored output
  -h, --help                Print help
  -V, --version             Print version
```

### 4.2 Example Workflows

```bash
# Find all errors with specific user
flog -f "level:error,user.id:12345" app.log

# Count 5xx errors per file
flog -f "status>=500" --count access-*.log

# Pretty print warnings
flog -f "level:warn" -o pretty app.log

# Extract specific fields
flog -f "level:error" -F "timestamp,message,stack" app.log

# Chain with other tools
flog -f "status:500" access.log | flog -f "path~=/api/users" -

# Use regex for complex patterns
flog -f "message~=timeout.*retry|connection.*refused" app.log
```

---

## 5. File Structure

```
flog/
├── cmd/
│   └── flog/
│       └── main.go           # CLI entry point
├── internal/
│   ├── parser/
│   │   ├── parser.go         # Parser interface + LogEntry
│   │   ├── json.go           # JSON log parser
│   │   ├── keyvalue.go       # Key-value parser
│   │   ├── auto.go           # Auto-detection
│   │   ├── reader.go         # Streaming file reader
│   │   └── parser_test.go    # Parser tests
│   ├── filter/
│   │   ├── filter.go         # Filter conditions + Matcher
│   │   ├── convert.go        # Type conversion utilities
│   │   ├── query.go          # Query DSL parser
│   │   ├── parallel.go       # Parallel processing
│   │   └── filter_test.go    # Filter tests
│   └── output/
│       └── output.go         # Output formatters
├── testdata/
│   └── sample.log            # Test data
├── docs/
│   └── LLD.md                # This document
├── go.mod
├── go.sum
├── .gitignore
└── README.md
```

---

## 6. Performance Considerations

### 6.1 Memory Management
- **Streaming:** Never load entire file; process line by line
- **Object Pooling:** Reuse `LogEntry` objects via `sync.Pool`
- **Buffer Reuse:** 64KB read buffers

### 6.2 CPU Optimization
- **Parallel Processing:** Worker pool sized to CPU cores
- **Regex Caching:** Compile regex patterns once, store in `sync.Map`
- **Short-circuit Evaluation:** Stop evaluating AND conditions on first false

### 6.3 I/O Optimization
- **Large Buffers:** 64KB+ read buffers
- **Gzip Support:** Transparent decompression of .gz files
- **Context Cancellation:** Graceful shutdown on Ctrl+C

---

## 7. Testing Strategy

### Unit Tests
- Parser: Various JSON/KV formats
- Filter: All operator types
- Query DSL: Grammar edge cases

### Benchmarks
```go
func BenchmarkJSONParser(b *testing.B) { ... }
func BenchmarkKeyValueParser(b *testing.B) { ... }
func BenchmarkMatcher(b *testing.B) { ... }
func BenchmarkQueryParser(b *testing.B) { ... }
```

Run benchmarks:
```bash
go test -bench=. ./...
```

---

## 8. Future Enhancements

- [ ] Config file support (~/.flogrc)
- [ ] Saved filter aliases
- [ ] Time range filters (--since, --until)
- [ ] Aggregation mode (group by field)
- [ ] Watch mode (tail -f equivalent)
- [ ] Multi-line log support
- [ ] Field type inference and casting
- [ ] JSON Lines streaming output
