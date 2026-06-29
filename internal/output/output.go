package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/imchristianrojas/logsift/internal/aggregator"
)

// topPaths is how many of the most-requested paths to show in the human
// formats (text/table). JSON always includes every path.
const topPaths = 10

// Write renders stats to w in the requested format ("table", "text", or "json").
func Write(w io.Writer, stats *aggregator.Stats, format string) error {
	switch format {
	case "json":
		return writeJSON(w, stats)
	case "text":
		return writeText(w, stats)
	case "table":
		return writeTable(w, stats)
	default:
		return fmt.Errorf("unknown format: %q", format)
	}
}

// writeJSON emits the full Stats as indented JSON.
func writeJSON(w io.Writer, stats *aggregator.Stats) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(stats)
}

// writeText emits a plain, line-oriented summary.
func writeText(w io.Writer, stats *aggregator.Stats) error {
	if _, err := fmt.Fprintf(w, "Total lines: %d\nBad lines:   %d\nTotal bytes: %d\n",
		stats.TotalLines, stats.BadLines, stats.TotalBytes); err != nil {
		return err
	}

	fmt.Fprintln(w, "\nStatus codes:")
	for _, sc := range sortedStatus(stats.StatusCounts) {
		fmt.Fprintf(w, "  %d: %d\n", sc.code, sc.count)
	}

	paths := topPathList(stats.PathCounts, topPaths)
	if len(paths) > 0 {
		fmt.Fprintf(w, "\nTop paths:\n")
		for _, p := range paths {
			fmt.Fprintf(w, "  %s: %d\n", p.path, p.count)
		}
	}
	return nil
}

// writeTable emits aligned columns using tabwriter.
func writeTable(w io.Writer, stats *aggregator.Stats) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	fmt.Fprintf(tw, "METRIC\tVALUE\n")
	fmt.Fprintf(tw, "Total lines\t%d\n", stats.TotalLines)
	fmt.Fprintf(tw, "Bad lines\t%d\n", stats.BadLines)
	fmt.Fprintf(tw, "Total bytes\t%d\n", stats.TotalBytes)

	fmt.Fprintf(tw, "\nSTATUS\tCOUNT\n")
	for _, sc := range sortedStatus(stats.StatusCounts) {
		fmt.Fprintf(tw, "%d\t%d\n", sc.code, sc.count)
	}

	paths := topPathList(stats.PathCounts, topPaths)
	if len(paths) > 0 {
		fmt.Fprintf(tw, "\nPATH\tCOUNT\n")
		for _, p := range paths {
			fmt.Fprintf(tw, "%s\t%d\n", p.path, p.count)
		}
	}
	return tw.Flush()
}

type statusCount struct {
	code  int
	count int
}

// sortedStatus returns status codes in ascending numeric order for stable output.
func sortedStatus(m map[int]int) []statusCount {
	out := make([]statusCount, 0, len(m))
	for code, count := range m {
		out = append(out, statusCount{code, count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].code < out[j].code })
	return out
}

type pathCount struct {
	path  string
	count int
}

// topPathList returns the n most-requested paths, busiest first. Ties are broken
// by path name so the output is deterministic.
func topPathList(m map[string]int, n int) []pathCount {
	out := make([]pathCount, 0, len(m))
	for path, count := range m {
		out = append(out, pathCount{path, count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].count != out[j].count {
			return out[i].count > out[j].count
		}
		return out[i].path < out[j].path
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}
