package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/ishk9/flog/internal/filter"
	"github.com/ishk9/flog/internal/output"
	"github.com/ishk9/flog/internal/parser"
)

// Version info - injected by GoReleaser via ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Config holds the CLI configuration.
type Config struct {
	Filter     string
	OutputFmt  string
	Fields     string
	Count      bool
	Limit      int
	IgnoreCase bool
	Invert     bool
	Jobs       int
	Stats      bool
	NoColor    bool
	Files      []string
}

func main() {
	cfg := parseFlags()

	if len(cfg.Files) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no input files specified")
		fmt.Fprintln(os.Stderr, "Usage: flog [OPTIONS] <FILE>...")
		fmt.Fprintln(os.Stderr, "Try 'flog --help' for more information.")
		os.Exit(1)
	}

	if cfg.Filter == "" {
		fmt.Fprintln(os.Stderr, "Error: no filter specified")
		fmt.Fprintln(os.Stderr, "Usage: flog -f <FILTER> <FILE>...")
		os.Exit(1)
	}

	// Parse filter query
	queryParser := filter.NewQueryParser()
	chain, err := queryParser.Parse(cfg.Filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing filter: %v\n", err)
		os.Exit(1)
	}

	// Handle invert flag
	if cfg.Invert {
		// Wrap the chain in a NOT logic (implemented as NE check)
		// For simplicity, we'll handle this in the matching phase
	}

	// Setup context with cancellation for Ctrl+C
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Setup output
	stats := output.NewStats()
	formatter := createFormatter(cfg)
	writer := output.NewWriter(os.Stdout, formatter, stats)

	if cfg.Limit > 0 {
		writer.SetLimit(int64(cfg.Limit))
	}

	// Process files
	reader := parser.NewStreamReader()
	p := parser.NewAutoParser()

	for _, file := range cfg.Files {
		if err := processFile(ctx, cfg, file, reader, p, chain, writer, stats); err != nil {
			if err == context.Canceled {
				break
			}
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", file, err)
		}
	}

	stats.Finish()

	// Output results based on mode
	if cfg.Count {
		fmt.Println(stats.MatchedLines)
	} else if cfg.Stats {
		printStats(stats)
	}
}

func parseFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.Filter, "f", "", "Filter expression (required)")
	flag.StringVar(&cfg.Filter, "filter", "", "Filter expression (required)")
	flag.StringVar(&cfg.OutputFmt, "o", "raw", "Output format: raw|pretty|json|fields")
	flag.StringVar(&cfg.OutputFmt, "output", "raw", "Output format: raw|pretty|json|fields")
	flag.StringVar(&cfg.Fields, "F", "", "Comma-separated fields to output")
	flag.StringVar(&cfg.Fields, "fields", "", "Comma-separated fields to output")
	flag.BoolVar(&cfg.Count, "c", false, "Print match count only")
	flag.BoolVar(&cfg.Count, "count", false, "Print match count only")
	flag.IntVar(&cfg.Limit, "n", 0, "Limit output to first N matches")
	flag.IntVar(&cfg.Limit, "limit", 0, "Limit output to first N matches")
	flag.BoolVar(&cfg.IgnoreCase, "i", false, "Case-insensitive matching")
	flag.BoolVar(&cfg.IgnoreCase, "ignore-case", false, "Case-insensitive matching")
	flag.BoolVar(&cfg.Invert, "v", false, "Invert match (print non-matching)")
	flag.BoolVar(&cfg.Invert, "invert", false, "Invert match (print non-matching)")
	flag.IntVar(&cfg.Jobs, "j", runtime.NumCPU(), "Number of parallel workers")
	flag.IntVar(&cfg.Jobs, "jobs", runtime.NumCPU(), "Number of parallel workers")
	flag.BoolVar(&cfg.Stats, "stats", false, "Print filter statistics")
	flag.BoolVar(&cfg.NoColor, "no-color", false, "Disable colored output")

	// Custom usage
	flag.Usage = printUsage

	// Check for version flag before parsing
	for _, arg := range os.Args[1:] {
		if arg == "-V" || arg == "--version" {
			fmt.Printf("flog %s\n", version)
			os.Exit(0)
		}
		if arg == "-h" || arg == "--help" {
			printUsage()
			os.Exit(0)
		}
	}

	flag.Parse()
	cfg.Files = flag.Args()

	return cfg
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `flog - Fast Log Filter v%s

USAGE:
    flog [OPTIONS] <FILE>...

ARGUMENTS:
    <FILE>...    Log file(s) to filter (use - for stdin)

OPTIONS:
    -f, --filter <QUERY>     Filter expression (required)
    -o, --output <FORMAT>    Output format: raw|pretty|json|fields [default: raw]
    -F, --fields <FIELDS>    Comma-separated fields to output
    -c, --count              Print match count only
    -n, --limit <N>          Limit output to first N matches
    -i, --ignore-case        Case-insensitive matching
    -v, --invert             Invert match (print non-matching lines)
    -j, --jobs <N>           Number of parallel workers [default: CPU count]
        --stats              Print filter statistics
        --no-color           Disable colored output
    -h, --help               Print help
    -V, --version            Print version

FILTER SYNTAX:
    field:value              Equality match
    field!=value             Not equal
    field>value              Greater than
    field<value              Less than  
    field>=value             Greater than or equal
    field<=value             Less than or equal
    field~=pattern           Regex match
    field*=substring         Contains substring
    field?                   Field exists

    Combine with:
    ,                        AND (all must match)
    |                        OR (any can match)

EXAMPLES:
    # Find all error logs
    flog -f "level:error" app.log

    # Find errors with status >= 500
    flog -f "level:error,status>=500" app.log

    # Find errors or warnings
    flog -f "level:error|level:warn" app.log

    # Filter nested fields
    flog -f "user.profile.role:admin" events.log

    # Regex matching
    flog -f "message~=timeout.*retry" app.log

    # Count matches
    flog -f "status>=400" --count access.log

    # Pretty print with colors
    flog -f "level:error" -o pretty app.log

    # Select specific fields
    flog -f "level:error" -F "timestamp,message" app.log

    # Read from stdin
    cat app.log | flog -f "level:error" -

`, version)
}

func createFormatter(cfg *Config) output.Formatter {
	if cfg.Fields != "" {
		fields := strings.Split(cfg.Fields, ",")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		return output.NewFieldsFormatter(fields, cfg.OutputFmt == "json")
	}

	switch cfg.OutputFmt {
	case "pretty":
		return output.NewPrettyFormatter(!cfg.NoColor)
	case "json":
		return output.NewJSONFormatter()
	case "fields":
		return output.NewFieldsFormatter(nil, false)
	default:
		return output.NewRawFormatter()
	}
}

func processFile(
	ctx context.Context,
	cfg *Config,
	path string,
	reader *parser.StreamReader,
	p parser.Parser,
	chain *filter.FilterChain,
	writer *output.Writer,
	stats *output.Stats,
) error {
	lines, errs := reader.ReadLines(ctx, path)

	// Create matcher
	matcher := filter.NewMatcher(cfg.IgnoreCase)
	lineNum := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errs:
			if err != nil {
				return err
			}
		case line, ok := <-lines:
			if !ok {
				return nil
			}

			lineNum++
			stats.IncrTotal()

			entry, err := p.Parse(line, lineNum)
			if err != nil {
				stats.IncrErrors()
				continue
			}

			matched := matcher.Match(entry, chain)
			if cfg.Invert {
				matched = !matched
			}

			if matched {
				if !cfg.Count {
					if !writer.Write(entry) {
						// Limit reached
						parser.ReleaseEntry(entry)
						return nil
					}
				} else {
					stats.IncrMatched()
				}
			}

			parser.ReleaseEntry(entry)
		}
	}
}

func printStats(stats *output.Stats) {
	fmt.Printf("\n--- Filter Statistics ---\n")
	fmt.Printf("Total lines:   %d\n", stats.TotalLines)
	fmt.Printf("Matched lines: %d\n", stats.MatchedLines)
	fmt.Printf("Parse errors:  %d\n", stats.ParseErrors)
	fmt.Printf("Duration:      %v\n", stats.Duration)
	if stats.TotalLines > 0 {
		rate := float64(stats.TotalLines) / stats.Duration.Seconds()
		fmt.Printf("Rate:          %.0f lines/sec\n", rate)
	}
}
