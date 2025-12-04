package parser

import (
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"strings"
)

const (
	// DefaultBufferSize is the default buffer size for reading files.
	DefaultBufferSize = 64 * 1024 // 64KB

	// DefaultChunkSize is the default number of lines per chunk for parallel processing.
	DefaultChunkSize = 1000
)

// StreamReader reads log files line by line with efficient buffering.
type StreamReader struct {
	BufferSize int
}

// NewStreamReader creates a new streaming reader.
func NewStreamReader() *StreamReader {
	return &StreamReader{
		BufferSize: DefaultBufferSize,
	}
}

// ReadLines reads a file and sends lines to a channel.
// Supports regular files, gzip files, and stdin (when path is "-").
func (r *StreamReader) ReadLines(ctx context.Context, path string) (<-chan string, <-chan error) {
	lines := make(chan string, 1000)
	errs := make(chan error, 1)

	go func() {
		defer close(lines)
		defer close(errs)

		reader, cleanup, err := r.openReader(path)
		if err != nil {
			errs <- err
			return
		}
		defer cleanup()

		scanner := bufio.NewScanner(reader)
		buf := make([]byte, r.BufferSize)
		scanner.Buffer(buf, r.BufferSize)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			case lines <- scanner.Text():
			}
		}

		if err := scanner.Err(); err != nil {
			errs <- err
		}
	}()

	return lines, errs
}

// ReadChunks reads a file and sends batches of lines for parallel processing.
func (r *StreamReader) ReadChunks(ctx context.Context, path string, chunkSize int) (<-chan []string, <-chan error) {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	chunks := make(chan []string, 10)
	errs := make(chan error, 1)

	go func() {
		defer close(chunks)
		defer close(errs)

		reader, cleanup, err := r.openReader(path)
		if err != nil {
			errs <- err
			return
		}
		defer cleanup()

		scanner := bufio.NewScanner(reader)
		buf := make([]byte, r.BufferSize)
		scanner.Buffer(buf, r.BufferSize)

		chunk := make([]string, 0, chunkSize)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			default:
			}

			chunk = append(chunk, scanner.Text())

			if len(chunk) >= chunkSize {
				select {
				case <-ctx.Done():
					errs <- ctx.Err()
					return
				case chunks <- chunk:
				}
				chunk = make([]string, 0, chunkSize)
			}
		}

		// Send remaining lines
		if len(chunk) > 0 {
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			case chunks <- chunk:
			}
		}

		if err := scanner.Err(); err != nil {
			errs <- err
		}
	}()

	return chunks, errs
}

// openReader opens the appropriate reader based on the path.
func (r *StreamReader) openReader(path string) (io.Reader, func(), error) {
	// Handle stdin
	if path == "-" {
		return os.Stdin, func() {}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}

	// Check for gzip
	if strings.HasSuffix(path, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			file.Close()
			return nil, nil, err
		}
		return gzReader, func() {
			gzReader.Close()
			file.Close()
		}, nil
	}

	return file, func() { file.Close() }, nil
}

// ReadAll reads all lines from a file into a slice.
// Use only for small files; prefer ReadLines for large files.
func (r *StreamReader) ReadAll(path string) ([]string, error) {
	ctx := context.Background()
	lines, errs := r.ReadLines(ctx, path)

	var result []string
	for line := range lines {
		result = append(result, line)
	}

	// Check for errors
	select {
	case err := <-errs:
		if err != nil && !errors.Is(err, io.EOF) {
			return result, err
		}
	default:
	}

	return result, nil
}

