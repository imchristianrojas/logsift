package parser

import (
	"testing"
)

func TestParse(t *testing.T) {
	line := `192.168.1.1 - - [10/Oct/2025:13:55:36 -0700] "GET /api/users HTTP/1.1" 200 1534 "https://example.com" "Mozilla/5.0"`

	entry, err := Parse(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Now assert each field is what you expect.
	// Check IP, Method, Path, Protocol, Status, Bytes, Referer, UserAgent.
	// Pattern for each:
	//   if entry.Field != expected {
	//       t.Errorf("Field = %v, want %v", entry.Field, expected)
	//   }

	if entry.IP != "192.168.1.1" {
		t.Errorf("IP = %v, want %v", entry.IP, "192.168.1.1")
	}
	if entry.Method != "GET" {
		t.Errorf("Method = %v, want %v", entry.Method, "GET")
	}
	if entry.Path != "/api/users" {
		t.Errorf("Path = %v, want %v", entry.Path, "/api/users")
	}
	if entry.Protocol != "HTTP/1.1" {
		t.Errorf("Protocol = %v, want %v", entry.Protocol, "HTTP/1.1")
	}
	if entry.Status != 200 {
		t.Errorf("Status = %v, want %v", entry.Status, 200)
	}
	if entry.Bytes != 1534 {
		t.Errorf("Bytes = %v, want %v", entry.Bytes, 1534)
	}
	if entry.Referer != "https://example.com" {
		t.Errorf("Referer = %v, want %v", entry.Referer, "https://example.com")
	}
	if entry.UserAgent != "Mozilla/5.0" {
		t.Errorf("UserAgent = %v, want %v", entry.UserAgent, "Mozilla/5.0")
	}
}

func TestParseMalformed(t *testing.T) {
	line := "this is not a valid log line"
	_, err := Parse(line)
	if err == nil {
		t.Error("expected error for malformed line, got nil")
	}
}
