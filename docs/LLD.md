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
    Raw      string                 // Original line
    Fields   map[string]any         // Parsed fields (flattened)
    LineNum  int                    // Line number in file
}

// Parser interface for different log formats
type Parser interface {
    Parse(line string) (*LogEntry, error)
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
}

// Matcher evaluates conditions against entries
type Matcher interface {
    Match(entry *LogEntry, chain *FilterChain) bool
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

# Mixed AND/OR (parentheses for grouping)
flog -f "(level:error|level:warn),status:500" access.log

# Nested fields
flog -f "user.profile.role:admin" access.log

# Regex matching
flog -f "message~=timeout.*retry" access.log

# Existence check
flog -f "error?" access.log  # Has 'error' field

# Negation
flog -f "level!=debug" access.log
```

**Query Grammar (BNF):**
```
query      → group ("," group)*
group      → condition ("|" condition)*
condition  → field operator value
field      → identifier ("." identifier)*
operator   → ":" | "=" | "!=" | ">" | "<" | ">=" | "<=" | "~=" | "*=" | "?"
value      → string | number | boolean
```

### 3.4 Streaming Reader (`internal/parser/reader.go`)

**Responsibility:** Read large files without memory bloat

```go
// StreamReader reads files line by line
type StreamReader struct {
    bufferSize int  // Default: 64KB per line buffer
}

func (r *StreamReader) Read(path string) <-chan string {
    // Returns channel that yields lines
    // Supports: regular files, gzip, stdin
}

// For parallel processing
func (r *StreamReader) ReadChunks(path string, chunkSize int) <-chan []string {
    // Returns channel of line batches for worker pools
}
```

### 3.5 Parallel Processor (`internal/filter/parallel.go`)

**Strategy:** Fan-out/fan-in with worker pools

```go
type ParallelFilter struct {
    Workers    int          // Default: runtime.NumCPU()
    ChunkSize  int          // Lines per chunk (default: 1000)
}

func (p *ParallelFilter) Filter(
    input <-chan []string,
    chain *FilterChain,
) <-chan *LogEntry {
    // 1. Spawn N workers
    // 2. Each worker parses + filters a chunk
    // 3. Results merged into output channel
}
```

**Pipeline Flow:**
```
File → [Chunk Reader] → [Worker Pool] → [Merger] → Output
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
    Format(entry *LogEntry) string
}

// Implementations:
// - RawFormatter    → Original line
// - PrettyFormatter → Colorized, indented JSON
// - JSONFormatter   → Compact JSON
// - FieldsFormatter → Only selected fields

type OutputMode int
const (
    ModeLines  OutputMode = iota  // Print matching lines
    ModeCount                     // Print count only
    ModeStats                     // Print field statistics
    ModeFirst                     // Print first N matches
)
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
      --stats               Print field statistics
  -h, --help                Print help
  -V, --version             Print version

Examples:
  flog -f "level:error" app.log
  flog -f "status>=400,method:POST" access.log
  flog -f "user.id:123|user.id:456" --output pretty events.json
  cat app.log | flog -f "error?" -
  flog -f "level:error" --count *.log
```

### 4.2 Example Workflows

```bash
# Find all errors with specific user
flog -f "level:error,user.id:12345" app.log

# Count 5xx errors per file
flog -f "status>=500" --count access-*.log

# Pretty print warnings from last hour (with jq pre-filter)
cat app.log | jq -c 'select(.timestamp > "2024-01-01T12:00:00")' | flog -f "level:warn" -o pretty -

# Extract specific fields
flog -f "level:error" -F "timestamp,message,stack" app.log

# Chain with other tools
flog -f "status:500" access.log | flog -f "path~=/api/users" -
```

---

## 5. Data Structures

### 5.1 Core Types

```go
// Entry pool for memory efficiency
var entryPool = sync.Pool{
    New: func() interface{} {
        return &LogEntry{
            Fields: make(map[string]any, 16),
        }
    },
}

// Compiled query for reuse
type CompiledQuery struct {
    Chain       *FilterChain
    RegexCache  map[string]*regexp.Regexp
}
```

### 5.2 Result Statistics

```go
type Stats struct {
    TotalLines   int64
    MatchedLines int64
    ParseErrors  int64
    Duration     time.Duration
    FieldCounts  map[string]int64  // For --stats mode
}
```

---

## 6. Performance Considerations

### 6.1 Memory Management
- **Streaming:** Never load entire file; process line by line
- **Object Pooling:** Reuse `LogEntry` objects via `sync.Pool`
- **Buffer Reuse:** Reuse byte buffers for parsing

### 6.2 CPU Optimization
- **Parallel Processing:** Worker pool sized to CPU cores
- **Regex Caching:** Compile regex patterns once
- **Short-circuit Evaluation:** Stop evaluating AND conditions on first false
- **SIMD (future):** Use SIMD for string matching where applicable

### 6.3 I/O Optimization
- **Large Buffers:** 64KB+ read buffers
- **Async I/O:** Read ahead while processing
- **Memory-mapped Files (optional):** For random access patterns

### 6.4 Benchmarks Target
| File Size | Target Time |
|-----------|-------------|
| 100 MB    | < 1 second  |
| 1 GB      | < 5 seconds |
| 10 GB     | < 30 seconds|

---

## 7. File Structure

```
flog/
├── cmd/
│   └── flog/
│       └── main.go           # CLI entry point
├── internal/
│   ├── parser/
│   │   ├── parser.go         # Parser interface
│   │   ├── json.go           # JSON log parser
│   │   ├── keyvalue.go       # Key-value parser
│   │   ├── auto.go           # Auto-detection
│   │   └── reader.go         # Streaming file reader
│   ├── filter/
│   │   ├── condition.go      # Filter conditions
│   │   ├── chain.go          # Filter chain (AND/OR)
│   │   ├── matcher.go        # Matching logic
│   │   ├── query.go          # Query DSL parser
│   │   └── parallel.go       # Parallel processing
│   └── output/
│       ├── formatter.go      # Output interface
│       ├── raw.go            # Raw output
│       ├── pretty.go         # Pretty printed
│       ├── json.go           # JSON output
│       └── stats.go          # Statistics output
├── go.mod
├── go.sum
├── README.md
└── LLD.md
```

---

## 8. Dependencies

```go
// go.mod
require (
    github.com/spf13/cobra v1.8.0    // CLI framework
    github.com/fatih/color v1.16.0   // Terminal colors
    github.com/json-iterator/go v1.1.12  // Fast JSON parsing
)
```

---

## 9. Future Enhancements

- [ ] Config file support (~/.flogrc)
- [ ] Saved filter aliases
- [ ] Time range filters (--since, --until)
- [ ] Aggregation mode (group by field)
- [ ] Watch mode (tail -f equivalent)
- [ ] Compressed file support (.gz, .zst)
- [ ] Multi-line log support
- [ ] Field type inference and casting

---

## 10. Testing Strategy

### Unit Tests
- Parser: Various JSON/KV formats
- Filter: All operator types
- Query DSL: Grammar edge cases

### Integration Tests
- End-to-end CLI tests
- Large file performance tests
- Pipe/stdin handling

### Benchmarks
```go
func BenchmarkFilter1GB(b *testing.B) { ... }
func BenchmarkParallelVsSerial(b *testing.B) { ... }
```

