package pool

import (
	"bufio"
	"os"
	"sync"

	"github.com/imchristianrojas/logsift/internal/aggregator"
	"github.com/imchristianrojas/logsift/internal/parser"
)

func Run(path string, workers int, parse parser.ParseFunc) (*aggregator.Stats, error) { // Run processes the log file at the given path using a pool of worker goroutines and returns aggregated statistics.
	logFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer logFile.Close() // Ensure the log file is closed when we're done with it

	jobs := make(chan string, 1000)                  // Create a buffered channel to hold log lines for processing
	results := make(chan *aggregator.Stats, workers) // Create a buffered channel to hold parsed log entries

	var wg sync.WaitGroup // Create a WaitGroup to wait for all workers to finish

	// 1. START THE WORKERS
	// Launch `workers` goroutines. Each one:
	//   - has its OWN local aggregator.New()
	//   - ranges over `jobs`, parsing each line, Add or AddBadLine into its local stats
	//   - when the range loop ends (jobs closed+drained), sends its local stats to `results`
	//   - calls wg.Done() when finished
	// Remember wg.Add(1) before launching each one.

	// Start the worker goroutines
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() { //
			defer wg.Done() // Ensure that we call Done() when the goroutine finishes
			localStats := aggregator.New()
			for line := range jobs {
				entry, err := parse(line)
				if err != nil {
					localStats.AddBadLine()
				} else {
					localStats.Add(entry)
				}
			}
			results <- localStats // Send the local stats to the results channel when done
		}() // Launch the goroutine
	}

	// 2. PRODUCE THE JOBS
	// Scan the file, send each line into `jobs`.
	// When done scanning, close(jobs) so workers' range loops terminate.
	scanner := bufio.NewScanner(logFile)
	for scanner.Scan() {
		jobs <- scanner.Text() // Send each scanned line into the jobs channel for processing by workers
	}
	close(jobs)
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// 3. WAIT, THEN CLOSE RESULTS
	// We need wg.Wait() to know all workers finished and sent their stats.
	// But wg.Wait() blocks — so it has to run in its own goroutine that
	// closes `results` once everyone's done. Otherwise the merge loop below
	// would block forever waiting for a close that never comes.
	go func() {
		wg.Wait()
		close(results)
	}()

	// 4. MERGE
	// Range over `results` (terminates when closed in step 3),
	// merging each worker's stats into one final Stats.
	final := aggregator.New()
	for s := range results {
		final.Merge(s)
	}
	return final, nil
}
