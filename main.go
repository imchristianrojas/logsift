package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/imchristianrojas/logsift/internal/aggregator"
	"github.com/imchristianrojas/logsift/internal/output"
	"github.com/imchristianrojas/logsift/internal/parser"
	"github.com/imchristianrojas/logsift/internal/pool"
)

func main() {

	file := flag.String("file", "", "path to log file (required)")
	workers := flag.Int("workers", 0, "number of workers goroutines to use (default: runtime.NumCPU())") //NumCPU returns the number of logical CPUs usable by the current process.
	format := flag.String("format", "table", "output format (default: table)")

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

	// RunChunked is the large-file path: it reads the file in big byte blocks
	// and parses them across `workers` goroutines. See internal/pool/largefile.go.
	stats, err := pool.RunChunked(*file, *workers)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if err := output.Write(os.Stdout, stats, *format); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func runSequential(path string) (*aggregator.Stats, error) {
	logFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}
	defer logFile.Close()

	stats := aggregator.New()
	scanner := bufio.NewScanner(logFile)

	for scanner.Scan() {
		line := scanner.Text()
		entry, err := parser.Parse(line)
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
