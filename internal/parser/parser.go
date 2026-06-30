package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var logPattern = regexp.MustCompile(`^([^ ]+) [^ ]+ [^ ]+ \[(.*?)\] "(\S+) (\S+) (\S+)" (\d+) (\d+) "(.*?)" "(.*?)"$`)

type LogEntry struct {
	IP        string
	Timestamp time.Time
	Method    string
	Path      string
	Protocol  string
	Status    int
	Bytes     int
	Referer   string
	UserAgent string
}

// ParseFunc is the shape every parser shares. Both Parse (combined) and
// ParseJSON satisfy it, so main.go can pick one at runtime and hand it to the
// worker pool — the seam that lets us A/B-benchmark regex vs JSON over the
// same machinery.
type ParseFunc func(line string) (*LogEntry, error)

func parseTimestamp(timestamp string) (time.Time, error) {
	layout := "02/Jan/2006:15:04:05 -0700"
	return time.Parse(layout, timestamp) // Parse the timestamp string into a time.Time object
}

func Parse(line string) (*LogEntry, error) {
	matches := logPattern.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("line does not match log pattern")
	}
	ts, err := parseTimestamp(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}
	status, err := strconv.Atoi(matches[6])
	if err != nil {
		return nil, fmt.Errorf("invalid status code: %w", err)
	}
	bytes, err := strconv.Atoi(matches[7])
	if err != nil {
		return nil, fmt.Errorf("invalid bytes: %w", err)
	}

	return &LogEntry{
		IP:        matches[1],
		Timestamp: ts,
		Method:    matches[3],
		Path:      matches[4],
		Protocol:  matches[5],
		Status:    status,
		Bytes:     bytes,
		Referer:   matches[8],
		UserAgent: matches[9],
	}, nil
}
