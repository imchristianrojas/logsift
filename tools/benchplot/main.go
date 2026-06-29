// Command benchplot turns `go test -bench` output into a chart.
//
// It reads benchmark results on stdin, prints a Unicode bar chart to stdout,
// and optionally writes a PNG (-png). It draws the PNG with nothing but the Go
// standard library (image, image/png) so the repo stays dependency-free.
//
// Usage:
//
//	go test -bench=. -run=^$ -count=1 . | go run ./tools/benchplot -png bench.png
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// result is one benchmark's parsed numbers.
type result struct {
	name  string  // cleaned name, e.g. "Sequential"
	nsOp  float64 // nanoseconds per operation
	bOp   int64   // bytes per op (0 if absent)
	alloc int64   // allocs per op (0 if absent)
}

func main() {
	pngPath := flag.String("png", "", "also write a PNG bar chart to this path")
	title := flag.String("title", "logsift benchmarks", "chart title")
	barWidth := flag.Int("width", 44, "width of the ASCII bars in characters")
	flag.Parse()

	results, err := parse(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "benchplot:", err)
		os.Exit(1)
	}
	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "benchplot: no benchmark lines found on stdin")
		os.Exit(1)
	}

	fmt.Print(renderASCII(results, *title, *barWidth))

	if *pngPath != "" {
		if err := writePNG(*pngPath, results, *title); err != nil {
			fmt.Fprintln(os.Stderr, "benchplot:", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "benchplot: wrote %s\n", *pngPath)
	}
}

// parse scans `go test -bench` output and pulls out one result per Benchmark
// line. A line looks like:
//
//	BenchmarkChunked-16   5   219174020 ns/op   393389081 B/op   1990661 allocs/op
func parse(r *os.File) ([]result, error) {
	var out []result
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 4 || !strings.HasPrefix(fields[0], "Benchmark") {
			continue
		}

		res := result{name: cleanName(fields[0])}
		// Each metric is a (value, unit) pair; find the units we care about and
		// read the number immediately before them.
		for i := 1; i < len(fields); i++ {
			switch fields[i] {
			case "ns/op":
				res.nsOp, _ = strconv.ParseFloat(fields[i-1], 64)
			case "B/op":
				res.bOp, _ = strconv.ParseInt(fields[i-1], 10, 64)
			case "allocs/op":
				res.alloc, _ = strconv.ParseInt(fields[i-1], 10, 64)
			}
		}
		if res.nsOp > 0 {
			out = append(out, res)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	// Slowest first so the chart reads top-to-bottom as "worst to best".
	sort.SliceStable(out, func(i, j int) bool { return out[i].nsOp > out[j].nsOp })
	return out, nil
}

// cleanName strips the "Benchmark" prefix and the "-N" GOMAXPROCS suffix that
// `go test` appends, then keeps only the leaf of a sub-benchmark path, e.g.
// "BenchmarkChunked-16" -> "Chunked" and
// "BenchmarkChunkedSizes/16MiB-16" -> "16MiB".
func cleanName(s string) string {
	s = strings.TrimPrefix(s, "Benchmark")
	if i := strings.LastIndexByte(s, '-'); i >= 0 {
		if _, err := strconv.Atoi(s[i+1:]); err == nil {
			s = s[:i]
		}
	}
	if i := strings.LastIndexByte(s, '/'); i >= 0 {
		s = s[i+1:]
	}
	return s
}

// slowest returns the largest ns/op, used as the baseline for bar scaling and
// for speedup ratios.
func slowest(rs []result) float64 {
	max := 0.0
	for _, r := range rs {
		if r.nsOp > max {
			max = r.nsOp
		}
	}
	return max
}

// humanTime turns nanoseconds into a compact human string.
func humanTime(ns float64) string {
	switch {
	case ns >= 1e9:
		return fmt.Sprintf("%.3f s", ns/1e9)
	case ns >= 1e6:
		return fmt.Sprintf("%.1f ms", ns/1e6)
	case ns >= 1e3:
		return fmt.Sprintf("%.1f us", ns/1e3)
	default:
		return fmt.Sprintf("%.0f ns", ns)
	}
}

// renderASCII builds the terminal/Markdown bar chart.
func renderASCII(rs []result, title string, width int) string {
	base := slowest(rs)

	// Pad names so the bars line up.
	nameW := 0
	for _, r := range rs {
		if len(r.name) > nameW {
			nameW = len(r.name)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", title)
	for _, r := range rs {
		filled := max(int(r.nsOp/base*float64(width)+0.5), 1)
		bar := strings.Repeat("█", filled) + strings.Repeat("·", width-filled)
		fmt.Fprintf(&b, "  %-*s  %s  %9s  (%.1fx)\n",
			nameW, r.name, bar, humanTime(r.nsOp), base/r.nsOp)
	}
	return b.String()
}
