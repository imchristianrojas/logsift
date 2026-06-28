package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
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

}
