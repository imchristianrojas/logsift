package pool

import (
	"bytes"
	"io"
	"os"
	"sync"

	"github.com/imchristianrojas/logsift/internal/aggregator"
	"github.com/imchristianrojas/logsift/internal/parser"
)

// defaultChunkSize is how many bytes we try to read from the file at a time.
// 1 MiB is large enough that the per-block channel overhead is negligible
// compared to the parsing work, while staying small enough to keep memory
// usage flat regardless of how big the file is.
const defaultChunkSize = 1 << 20 // 1 MiB

// RunChunked processes the log file at path using a pool of workers, but unlike
// Run it sends whole blocks of bytes (many lines at once) over the channel
// instead of one line at a time. For very large files this turns millions of
// tiny channel sends into a few thousand big ones, which is a large speedup.
func RunChunked(path string, workers int) (*aggregator.Stats, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return runChunked(f, workers, defaultChunkSize)
}

// runChunked is the testable core: it reads from any io.Reader and takes an
// explicit chunkSize so tests can use a tiny size to force lines to be split
// across block boundaries (the part that's easy to get wrong).
func runChunked(r io.Reader, workers int, chunkSize int) (*aggregator.Stats, error) {
	jobs := make(chan []byte, workers)               // blocks of complete lines
	results := make(chan *aggregator.Stats, workers) // one final Stats per worker

	var wg sync.WaitGroup

	// 1. START THE WORKERS
	// Each worker owns its own aggregator (no shared state), pulls byte blocks
	// off `jobs`, splits each block into lines, and parses them.
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := aggregator.New()
			for block := range jobs {
				processBlock(block, local)
			}
			results <- local
		}()
	}

	// 2. PRODUCE THE BLOCKS
	// We read fixed-size chunks, but a chunk almost never ends exactly on a
	// newline, so each read leaves a partial trailing line. We send everything
	// up to and including the last newline as a block, and carry the leftover
	// bytes forward to be glued onto the front of the next read.
	readErr := readBlocks(r, chunkSize, jobs)
	close(jobs)

	// 3. WAIT, THEN CLOSE RESULTS
	// Once every worker has finished draining `jobs` and pushed its Stats,
	// close `results` so the merge loop below terminates.
	go func() {
		wg.Wait()
		close(results)
	}()

	// 4. MERGE
	final := aggregator.New()
	for s := range results {
		final.Merge(s)
	}

	if readErr != nil {
		return nil, readErr
	}
	return final, nil
}

// readBlocks reads r in chunkSize-byte reads and sends blocks of complete lines
// into jobs. The trailing partial line from one read is prepended to the next.
func readBlocks(r io.Reader, chunkSize int, jobs chan<- []byte) error {
	buf := make([]byte, chunkSize)
	var leftover []byte // partial line carried over from the previous read

	for {
		n, err := r.Read(buf)
		if n > 0 {
			// Glue the previous partial line onto what we just read.
			data := append(leftover, buf[:n]...)

			if nl := bytes.LastIndexByte(data, '\n'); nl >= 0 {
				// Everything up to and including the last '\n' is a set of
				// complete lines. Copy it into its own slice before sending,
				// because `buf` gets overwritten on the next read and would
				// otherwise corrupt the block a worker is still reading.
				block := make([]byte, nl+1)
				copy(block, data[:nl+1])
				jobs <- block

				// Keep the bytes after the last newline as the next leftover.
				leftover = append([]byte(nil), data[nl+1:]...)
			} else {
				// No newline in the whole buffer (a single line longer than
				// chunkSize): keep accumulating.
				leftover = append([]byte(nil), data...)
			}
		}

		if err == io.EOF {
			// Flush the final line, which has no trailing newline.
			if len(leftover) > 0 {
				jobs <- leftover
			}
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// processBlock splits a block of bytes into individual lines and folds each one
// into stats.
func processBlock(block []byte, stats *aggregator.Stats) {
	for len(block) > 0 {
		var line []byte
		if i := bytes.IndexByte(block, '\n'); i >= 0 {
			line, block = block[:i], block[i+1:]
		} else {
			line, block = block, nil
		}

		// Tolerate Windows-style "\r\n" line endings.
		line = bytes.TrimSuffix(line, []byte("\r"))
		if len(line) == 0 {
			continue
		}

		entry, err := parser.Parse(string(line))
		if err != nil {
			stats.AddBadLine()
		} else {
			stats.Add(entry)
		}
	}
}
