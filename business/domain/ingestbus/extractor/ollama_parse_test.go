package extractor

import (
	"fmt"
	"strings"
	"testing"
)

// TestData is a simple struct for testing JSON unmarshaling.
type TestData struct {
	A int    `json:"a"`
	B string `json:"b,omitempty"`
}

func TestParseOllamaJSON_HappyPath(t *testing.T) {
	raw := `{"a":1}`
	var dest TestData
	err := parseOllamaJSON(raw, &dest)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if dest.A != 1 {
		t.Errorf("expected A=1, got %d", dest.A)
	}
}

func TestParseOllamaJSON_EmptyString(t *testing.T) {
	raw := ""
	var dest TestData
	err := parseOllamaJSON(raw, &dest)
	if err == nil {
		t.Fatal("expected error for empty string, got nil")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("error should contain 'empty response', got %v", err)
	}
}

func TestParseOllamaJSON_WhitespaceOnly(t *testing.T) {
	raw := "   \n  \t  "
	var dest TestData
	err := parseOllamaJSON(raw, &dest)
	if err == nil {
		t.Fatal("expected error for whitespace-only string, got nil")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("error should contain 'empty response', got %v", err)
	}
}

func TestParseOllamaJSON_MarkdownFenceWithJsonTag(t *testing.T) {
	raw := "```json\n{\"a\":2}\n```"
	var dest TestData
	err := parseOllamaJSON(raw, &dest)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if dest.A != 2 {
		t.Errorf("expected A=2, got %d", dest.A)
	}
}

func TestParseOllamaJSON_MarkdownFenceWithoutTag(t *testing.T) {
	raw := "```\n{\"a\":3}\n```"
	var dest TestData
	err := parseOllamaJSON(raw, &dest)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if dest.A != 3 {
		t.Errorf("expected A=3, got %d", dest.A)
	}
}

func TestParseOllamaJSON_MarkdownFenceWithExtraWhitespace(t *testing.T) {
	raw := "  ```json  \n{\"a\":4}\n```  "
	var dest TestData
	err := parseOllamaJSON(raw, &dest)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if dest.A != 4 {
		t.Errorf("expected A=4, got %d", dest.A)
	}
}

func TestParseOllamaJSON_InvalidJSON(t *testing.T) {
	raw := "not json at all"
	var dest TestData
	err := parseOllamaJSON(raw, &dest)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "unmarshal") {
		t.Errorf("error should contain 'unmarshal', got %v", err)
	}
	if !strings.Contains(errMsg, "raw=") {
		t.Errorf("error should contain 'raw=' with snippet, got %v", err)
	}
}

func TestParseOllamaJSON_InvalidJSONIncludesTruncatedRaw(t *testing.T) {
	// Create a raw response longer than 200 characters.
	raw := fmt.Sprintf("invalid json: %s", strings.Repeat("x", 250))
	var dest TestData
	err := parseOllamaJSON(raw, &dest)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	errMsg := err.Error()
	// The error should include a truncated version of raw (first 200 chars).
	if !strings.Contains(errMsg, "raw=") {
		t.Errorf("error should contain 'raw=' snippet, got %v", err)
	}
	// Verify that the snippet is truncated (contains only first 200 chars).
	if strings.Contains(errMsg, raw) {
		// The full raw should not be in the error if it's longer than 200 chars.
		t.Errorf("error should not contain full raw response (> 200 chars), got %v", err)
	}
}

func TestParseOllamaJSON_NestedJSON(t *testing.T) {
	raw := `{"a":5,"b":"nested"}`
	var dest TestData
	err := parseOllamaJSON(raw, &dest)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if dest.A != 5 {
		t.Errorf("expected A=5, got %d", dest.A)
	}
	if dest.B != "nested" {
		t.Errorf("expected B='nested', got %q", dest.B)
	}
}

func TestParseOllamaJSON_MarkdownFenceMultilineJSON(t *testing.T) {
	raw := "```json\n{\n  \"a\": 6,\n  \"b\": \"multiline\"\n}\n```"
	var dest TestData
	err := parseOllamaJSON(raw, &dest)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if dest.A != 6 {
		t.Errorf("expected A=6, got %d", dest.A)
	}
	if dest.B != "multiline" {
		t.Errorf("expected B='multiline', got %q", dest.B)
	}
}

func TestParseOllamaJSON_MalformedJSONWithinFence(t *testing.T) {
	raw := "```json\n{invalid json}\n```"
	var dest TestData
	err := parseOllamaJSON(raw, &dest)
	if err == nil {
		t.Fatal("expected error for malformed JSON within fence, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("error should contain 'unmarshal', got %v", err)
	}
}
