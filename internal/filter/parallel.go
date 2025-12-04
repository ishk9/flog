package filter

import (
	"context"
	"runtime"
	"sync"

	"github.com/ishk9/flog/internal/parser"
)

// DefaultWorkers is the default number of parallel workers.
var DefaultWorkers = runtime.NumCPU()

// ParallelFilter performs parallel filtering of log entries.
type ParallelFilter struct {
	Workers   int
	ChunkSize int
	parser    parser.Parser
	matcher   *Matcher
}

// NewParallelFilter creates a new parallel filter.
func NewParallelFilter(p parser.Parser, ignoreCase bool) *ParallelFilter {
	return &ParallelFilter{
		Workers:   DefaultWorkers,
		ChunkSize: parser.DefaultChunkSize,
		parser:    p,
		matcher:   NewMatcher(ignoreCase),
	}
}

// FilterResult contains the result of filtering a single line.
type FilterResult struct {
	Entry   *parser.LogEntry
	Matched bool
	Err     error
}

// Filter processes lines from a channel and returns matching entries.
func (pf *ParallelFilter) Filter(
	ctx context.Context,
	lines <-chan string,
	chain *FilterChain,
) <-chan *parser.LogEntry {
	results := make(chan *parser.LogEntry, pf.Workers*2)

	var wg sync.WaitGroup
	lineNum := 0
	lineNumMu := sync.Mutex{}

	// Start workers
	for i := 0; i < pf.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case line, ok := <-lines:
					if !ok {
						return
					}

					// Get line number
					lineNumMu.Lock()
					lineNum++
					currentLineNum := lineNum
					lineNumMu.Unlock()

					// Parse and filter
					entry, err := pf.parser.Parse(line, currentLineNum)
					if err != nil {
						continue // Skip unparseable lines
					}

					if pf.matcher.Match(entry, chain) {
						select {
						case <-ctx.Done():
							parser.ReleaseEntry(entry)
							return
						case results <- entry:
						}
					} else {
						parser.ReleaseEntry(entry)
					}
				}
			}
		}()
	}

	// Close results when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// FilterChunks processes chunks of lines in parallel.
func (pf *ParallelFilter) FilterChunks(
	ctx context.Context,
	chunks <-chan []string,
	chain *FilterChain,
	startLineNum int,
) <-chan *parser.LogEntry {
	results := make(chan *parser.LogEntry, pf.Workers*2)

	var wg sync.WaitGroup
	lineOffset := startLineNum
	offsetMu := sync.Mutex{}

	// Start workers
	for i := 0; i < pf.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case chunk, ok := <-chunks:
					if !ok {
						return
					}

					// Get starting line number for this chunk
					offsetMu.Lock()
					chunkStart := lineOffset
					lineOffset += len(chunk)
					offsetMu.Unlock()

					// Process chunk
					for i, line := range chunk {
						entry, err := pf.parser.Parse(line, chunkStart+i)
						if err != nil {
							continue
						}

						if pf.matcher.Match(entry, chain) {
							select {
							case <-ctx.Done():
								parser.ReleaseEntry(entry)
								return
							case results <- entry:
							}
						} else {
							parser.ReleaseEntry(entry)
						}
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// SequentialFilter performs single-threaded filtering (for small files or ordered output).
type SequentialFilter struct {
	parser  parser.Parser
	matcher *Matcher
}

// NewSequentialFilter creates a new sequential filter.
func NewSequentialFilter(p parser.Parser, ignoreCase bool) *SequentialFilter {
	return &SequentialFilter{
		parser:  p,
		matcher: NewMatcher(ignoreCase),
	}
}

// Filter processes lines sequentially and returns matching entries.
func (sf *SequentialFilter) Filter(
	ctx context.Context,
	lines <-chan string,
	chain *FilterChain,
) <-chan *parser.LogEntry {
	results := make(chan *parser.LogEntry, 100)

	go func() {
		defer close(results)
		lineNum := 0

		for {
			select {
			case <-ctx.Done():
				return
			case line, ok := <-lines:
				if !ok {
					return
				}

				lineNum++
				entry, err := sf.parser.Parse(line, lineNum)
				if err != nil {
					continue
				}

				if sf.matcher.Match(entry, chain) {
					select {
					case <-ctx.Done():
						parser.ReleaseEntry(entry)
						return
					case results <- entry:
					}
				} else {
					parser.ReleaseEntry(entry)
				}
			}
		}
	}()

	return results
}

