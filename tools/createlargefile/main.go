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
	count := flag.Int("count", 500000, "number of log lines to generate")
	out := flag.String("out", "testdata/big.log", "output file path")
	seed := flag.Int64("seed", 42, "PRNG seed — fixed so the file is byte-identical across runs/machines")
	flag.Parse()

	// Seed a local PRNG (not the global one) so output is deterministic and
	// reproducible across machines — same -seed yields the same file. The
	// global rand is auto-seeded at startup, which would defeat that.
	r := rand.New(rand.NewSource(*seed))

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
	// Fixed base time (not time.Now()) so the file is reproducible — the actual
	// date is irrelevant for a synthetic benchmark log.
	ts := time.Date(2025, 10, 10, 13, 55, 36, 0, time.FixedZone("PDT", -7*3600))

	for i := 0; i < *count; i++ {
		ip := fmt.Sprintf("%d.%d.%d.%d", r.Intn(256), r.Intn(256), r.Intn(256), r.Intn(256))
		ts = ts.Add(time.Second)
		fmt.Fprintf(w, "%s - - [%s] \"%s %s HTTP/1.1\" %d %d \"%s\" \"%s\"\n",
			ip,
			ts.Format(layout),
			methods[r.Intn(len(methods))],
			paths[r.Intn(len(paths))],
			statuses[r.Intn(len(statuses))],
			r.Intn(50000),
			"https://example.com",
			"Mozilla/5.0",
		)
	}

	fmt.Printf("Wrote %d lines to %s\n", *count, *out)
}
