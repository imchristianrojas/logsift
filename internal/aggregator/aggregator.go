package aggregator

import (
	"github.com/imchristianrojas/logsift/internal/parser"
)

type Stats struct {
	TotalLines   int            `json:"total_lines"`
	BadLines     int            `json:"bad_lines"`
	TotalBytes   int64          `json:"total_bytes"`
	StatusCounts map[int]int    `json:"status_counts"` //status code -> count
	PathCounts   map[string]int `json:"path_counts"`   //path -> count
}

func New() *Stats { // New returns a new Stats instance with initialized maps
	return &Stats{
		StatusCounts: make(map[int]int),
		PathCounts:   make(map[string]int),
	}
}

func (s *Stats) Add(entry *parser.LogEntry) { // Add updates the statistics with a new log entry
	s.TotalLines++
	s.TotalBytes += int64(entry.Bytes)
	s.StatusCounts[entry.Status]++
	s.PathCounts[entry.Path]++
}

func (s *Stats) AddBadLine() { // AddBadLine increments the count of bad lines
	s.TotalLines++
	s.BadLines++
}
func (s *Stats) Merge(other *Stats) { // Merge combines another Stats instance into this one
	s.TotalLines += other.TotalLines
	s.BadLines += other.BadLines
	s.TotalBytes += other.TotalBytes
	for code, count := range other.StatusCounts {
		s.StatusCounts[code] += count
	}
	for path, count := range other.PathCounts {
		s.PathCounts[path] += count
	}

}
