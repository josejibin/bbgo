package output

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestPrintJSON(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"key": "value"}
	if err := PrintJSON(data); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify it's valid JSON
	var parsed map[string]string
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}
	if parsed["key"] != "value" {
		t.Errorf("expected key=value, got %v", parsed)
	}

	// Verify it's indented
	if !strings.Contains(output, "  ") {
		t.Error("expected indented JSON output")
	}
}

func TestTableOutput(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tbl := NewTable()
	tbl.Row("NAME", "AGE")
	tbl.Row("Alice", "30")
	tbl.Row("Bob", "25")
	tbl.Flush()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), output)
	}

	// Verify header
	if !strings.Contains(lines[0], "NAME") || !strings.Contains(lines[0], "AGE") {
		t.Errorf("expected header with NAME and AGE, got %q", lines[0])
	}

	// Verify data rows
	if !strings.Contains(lines[1], "Alice") || !strings.Contains(lines[1], "30") {
		t.Errorf("expected Alice/30 in row, got %q", lines[1])
	}
}
