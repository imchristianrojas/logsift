package parser

import (
	"encoding/json"
	"fmt"
	"time"
)

// jsonEntry mirrors the nested on-disk shape of a Caddy-style JSON log line.
// It's kept separate from LogEntry so our internal/output shape isn't chained
// to whatever the input log looks like. The `json:"..."` tags are matched at
// runtime via reflection — flexible, but the main reason JSON is slower here.
type jsonEntry struct {
	Ts      time.Time `json:"ts"` // RFC3339 string decoded straight into time.Time
	Status  int       `json:"status"`
	Size    int       `json:"size"`
	Request struct {
		RemoteIP string `json:"remote_ip"`
		Proto    string `json:"proto"`
		Method   string `json:"method"`
		URI      string `json:"uri"`
		// Caddy logs headers as arrays (headers can repeat); we take the first.
		Headers struct {
			UserAgent []string `json:"User-Agent"`
			Referer   []string `json:"Referer"`
		} `json:"headers"`
	} `json:"request"`
}

// ParseJSON decodes one nested JSON line into a LogEntry. Same signature as
// Parse, so it satisfies ParseFunc and drops into the worker pool unchanged.
func ParseJSON(line string) (*LogEntry, error) {
	var je jsonEntry
	// []byte(line) allocates a copy each call — the regex path skips this by
	// matching the string directly. A small but real part of the cost gap.
	if err := json.Unmarshal([]byte(line), &je); err != nil {
		return nil, fmt.Errorf("invalid json log line: %w", err)
	}

	// encoding/json is permissive: a valid object missing our fields decodes
	// fine with zero values. Reject those so a BadLine means the same thing on
	// both the JSON and regex paths.
	if je.Request.Method == "" || je.Status == 0 {
		return nil, fmt.Errorf("json log line missing required fields")
	}

	return &LogEntry{
		IP:        je.Request.RemoteIP,
		Timestamp: je.Ts,
		Method:    je.Request.Method,
		Path:      je.Request.URI,
		Protocol:  je.Request.Proto,
		Status:    je.Status,
		Bytes:     je.Size,
		Referer:   firstHeader(je.Request.Headers.Referer),
		UserAgent: firstHeader(je.Request.Headers.UserAgent),
	}, nil
}

func firstHeader(vals []string) string {
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}
