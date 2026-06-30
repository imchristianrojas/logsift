package parser

import "testing"

// One Caddy-style line; mirrors the same logical record as combinedLine below.
const jsonLine = `{"level":"info","ts":"2025-10-10T13:55:36Z","logger":"http.log.access","msg":"handled request","request":{"remote_ip":"192.168.1.1","remote_port":"54321","proto":"HTTP/1.1","method":"GET","host":"api.example.com","uri":"/api/users","headers":{"User-Agent":["Mozilla/5.0"],"Referer":["https://example.com"]}},"user_id":"42","duration":0.0153,"size":1534,"status":200}`

// combinedLine is the SAME record in combined log format, for a fair head-to-head.
const combinedLine = `192.168.1.1 - - [10/Oct/2025:13:55:36 -0700] "GET /api/users HTTP/1.1" 200 1534 "https://example.com" "Mozilla/5.0"`

func TestParseJSON(t *testing.T) {
	e, err := ParseJSON(jsonLine)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.IP != "192.168.1.1" {
		t.Errorf("IP = %v, want 192.168.1.1", e.IP)
	}
	if e.Method != "GET" {
		t.Errorf("Method = %v, want GET", e.Method)
	}
	if e.Path != "/api/users" {
		t.Errorf("Path = %v, want /api/users", e.Path)
	}
	if e.Status != 200 {
		t.Errorf("Status = %v, want 200", e.Status)
	}
	if e.Bytes != 1534 {
		t.Errorf("Bytes = %v, want 1534", e.Bytes)
	}
	if e.UserAgent != "Mozilla/5.0" {
		t.Errorf("UserAgent = %v, want Mozilla/5.0", e.UserAgent)
	}
}

func TestParseJSONBad(t *testing.T) {
	cases := map[string]string{
		"garbage":        "not json at all",
		"truncated":      `{"request":{"method":"GET"`,
		"missing fields": `{"level":"info","msg":"no request here"}`, // valid JSON, no method/status
	}
	for name, line := range cases {
		if _, err := ParseJSON(line); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

// BenchmarkParse* isolate the parser cost from I/O and concurrency: same
// logical record, parsed b.N times in memory. This is where the regex-vs-JSON
// (reflection) gap shows up cleanly.
func BenchmarkParseCombined(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := Parse(combinedLine); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseJSON(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := ParseJSON(jsonLine); err != nil {
			b.Fatal(err)
		}
	}
}
