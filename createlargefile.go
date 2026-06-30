package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
)

func main() {
	count := flag.Int("count", 1000000, "number of log lines to generate")
	out := flag.String("out", "testdata/big.log", "output file path")
	flag.Parse()

	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	defer f.Close()

	// Buffer the writes — writing a million lines with unbuffered
	// I/O is painfully slow (a syscall per write). bufio.Writer
	// batches them. MUST flush at the end (see note).
	w := bufio.NewWriter(f)
	defer w.Flush()

	// Pools to randomize over, so the aggregation has variety:
	paths := []string{"/", "/api/users", "/login", "/api/products", "/static/app.js", "/health"}
	methods := []string{"GET", "GET", "GET", "POST", "PUT"} // weighted toward GET
	statuses := []int{200, 200, 200, 200, 404, 500, 301}    // weighted toward 200

	for i := 0; i < *count; i++ {
		// Build ONE valid Nginx combined-format line and write it.
		// Format reminder:
		// IP - - [02/Jan/2006:15:04:05 -0700] "METHOD PATH HTTP/1.1" STATUS BYTES "REFERER" "UA"
		//
		// - random IP: fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), ...)
		// - timestamp: time.Now().Format("02/Jan/2006:15:04:05 -0700")
		//   (same layout string as your parser — they must match!)
		// - method, path, status: pick randomly from the pools above
		// - bytes: rand.Intn(some range)
		// - referer/UA: can be fixed strings, doesn't matter for benchmarking
		//
		// Write it with fmt.Fprintf(w, "...\n", ...) — don't forget the \n
	}

	fmt.Printf("Wrote %d lines to %s\n", *count, *out)
}
