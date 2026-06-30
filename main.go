package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/imchristianrojas/logsift/internal/aggregator"
	"github.com/imchristianrojas/logsift/internal/output"
	"github.com/imchristianrojas/logsift/internal/parser"
	"github.com/imchristianrojas/logsift/internal/pool"
)

// parseSize converts a human-friendly size like "256KiB", "1MiB", "4M", or a
// plain byte count like "1048576" into a number of bytes. Suffixes are binary
// (1 KiB = 1024). The "iB"/"B" tail is optional, so "4M" == "4MiB".
func parseSize(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size")
	}

	mult := 1
	// Strip an optional trailing "b"/"ib", then a unit letter.
	u := strings.ToLower(s)
	u = strings.TrimSuffix(u, "b")
	u = strings.TrimSuffix(u, "i")
	switch {
	case strings.HasSuffix(u, "k"):
		mult, u = 1<<10, strings.TrimSuffix(u, "k")
	case strings.HasSuffix(u, "m"):
		mult, u = 1<<20, strings.TrimSuffix(u, "m")
	case strings.HasSuffix(u, "g"):
		mult, u = 1<<30, strings.TrimSuffix(u, "g")
	}

	n, err := strconv.Atoi(strings.TrimSpace(u))
	if err != nil {
		return 0, fmt.Errorf("%q is not a valid size", s)
	}
	if n <= 0 {
		return 0, fmt.Errorf("size must be positive")
	}
	return n * mult, nil
}

func main() {

	file := flag.String("file", "", "path to log file (required)")
	workers := flag.Int("workers", 0, "number of workers goroutines to use (default: runtime.NumCPU())") //NumCPU returns the number of logical CPUs usable by the current process.
	format := flag.String("format", "table", "output format (default: table)")
	inputFormat := flag.String("input-format", "combined", "input log format: combined or json")
	chunkSize := flag.String("chunk-size", "1MiB", "block size for the chunked reader (e.g. 256KiB, 1MiB, 4MiB)")

	flag.Parse()

	if *file == "" {
		fmt.Fprintln(os.Stderr, "Error: --file is required")
		os.Exit(1)
	}

	if *workers <= 0 {
		*workers = runtime.NumCPU() //If workers is not specified or less than or equal to 0, set it to the number of logical CPUs
	}

	if *format != "table" && *format != "text" && *format != "json" {
		fmt.Fprintln(os.Stderr, "Error: --format must be 'table', 'text', or 'json'")
		os.Exit(1)
	}

	chunkBytes, err := parseSize(*chunkSize)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: invalid --chunk-size:", err)
		os.Exit(1)
	}

	// Pick the parser for the input format. Both satisfy parser.ParseFunc, so
	// the pool below is identical regardless of which one we hand it.
	var parse parser.ParseFunc
	switch *inputFormat {
	case "combined":
		parse = parser.Parse
	case "json":
		parse = parser.ParseJSON
	default:
		fmt.Fprintln(os.Stderr, "Error: --input-format must be 'combined' or 'json'")
		os.Exit(1)
	}

	// RunChunkedSize is the large-file path: it reads the file in fixed byte
	// blocks and parses them across `workers` goroutines. See
	// internal/pool/largefile.go.
	stats, err := pool.RunChunkedSize(*file, *workers, chunkBytes, parse)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if err := output.Write(os.Stdout, stats, *format); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func runSequential(path string, parse parser.ParseFunc) (*aggregator.Stats, error) {
	logFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}
	defer logFile.Close()

	stats := aggregator.New()
	scanner := bufio.NewScanner(logFile)

	for scanner.Scan() {
		line := scanner.Text()
		entry, err := parse(line)
		if err != nil {
			stats.AddBadLine()
			continue
		}
		stats.Add(entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading log file: %w", err)
	}

	return stats, nil
}
