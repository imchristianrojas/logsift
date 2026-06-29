package pool

import (
	"strings"
	"testing"
)

// Three good lines (statuses 200, 404, 200) and one unparseable line.
const sampleLog = `192.168.1.1 - - [10/Oct/2025:13:55:36 -0700] "GET /api/users HTTP/1.1" 200 1534 "https://example.com" "Mozilla/5.0"
10.0.0.5 - - [10/Oct/2025:13:55:37 -0700] "POST /login HTTP/1.1" 404 230 "-" "curl/7.68"
garbage line that should fail
192.168.1.2 - - [10/Oct/2025:13:55:38 -0700] "GET /api/users HTTP/1.1" 200 512 "-" "Mozilla/5.0"
`

func TestRunChunked_BoundaryAndWorkers(t *testing.T) {
	// Tiny chunk sizes force lines (and even single fields) to straddle block
	// boundaries, exercising the leftover-carry logic. Varying worker counts
	// checks that the merge is order-independent.
	chunkSizes := []int{1, 3, 7, 64, 1 << 20}
	workerCounts := []int{1, 2, 4}

	for _, cs := range chunkSizes {
		for _, w := range workerCounts {
			stats, err := runChunked(strings.NewReader(sampleLog), w, cs)
			if err != nil {
				t.Fatalf("chunkSize=%d workers=%d: unexpected error: %v", cs, w, err)
			}

			if stats.TotalLines != 4 {
				t.Errorf("chunkSize=%d workers=%d: TotalLines = %d, want 4", cs, w, stats.TotalLines)
			}
			if stats.BadLines != 1 {
				t.Errorf("chunkSize=%d workers=%d: BadLines = %d, want 1", cs, w, stats.BadLines)
			}
			if got := stats.TotalBytes; got != 1534+230+512 {
				t.Errorf("chunkSize=%d workers=%d: TotalBytes = %d, want %d", cs, w, got, 1534+230+512)
			}
			if got := stats.StatusCounts[200]; got != 2 {
				t.Errorf("chunkSize=%d workers=%d: StatusCounts[200] = %d, want 2", cs, w, got)
			}
			if got := stats.StatusCounts[404]; got != 1 {
				t.Errorf("chunkSize=%d workers=%d: StatusCounts[404] = %d, want 1", cs, w, got)
			}
		}
	}
}

// A line longer than the chunk size must still be assembled correctly across
// multiple reads before the next newline appears.
func TestRunChunked_NoTrailingNewline(t *testing.T) {
	input := strings.TrimRight(sampleLog, "\n") // drop the final newline
	stats, err := runChunked(strings.NewReader(input), 2, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalLines != 4 {
		t.Errorf("TotalLines = %d, want 4", stats.TotalLines)
	}
	if stats.BadLines != 1 {
		t.Errorf("BadLines = %d, want 1", stats.BadLines)
	}
}
