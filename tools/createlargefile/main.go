package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"
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

	// Same layout string as the parser (internal/parser/parser.go) — they must match.
	const layout = "02/Jan/2006:15:04:05 -0700"
	ts := time.Now()

	for i := 0; i < *count; i++ {
		ip := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256))
		ts = ts.Add(time.Second)
		fmt.Fprintf(w, "%s - - [%s] \"%s %s HTTP/1.1\" %d %d \"%s\" \"%s\"\n",
			ip,
			ts.Format(layout),
			methods[rand.Intn(len(methods))],
			paths[rand.Intn(len(paths))],
			statuses[rand.Intn(len(statuses))],
			rand.Intn(50000),
			"https://example.com",
			"Mozilla/5.0",
		)
	}

	fmt.Printf("Wrote %d lines to %s\n", *count, *out)
}
