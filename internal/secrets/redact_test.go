package secrets

import (
	"bytes"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	// Set a token for testing
	loadedToken = "super-secret-token-123"
	defer func() { loadedToken = "" }()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "token replacement",
			input: "Authorization: Basic super-secret-token-123",
			want:  "Authorization: Basic [REDACTED]",
		},
		{
			name:  "pattern: token=value",
			input: "token=abc123 other text",
			want:  "token=[REDACTED] other text",
		},
		{
			name:  "pattern: password: value",
			input: "password: mysecret other",
			want:  "password=[REDACTED] other",
		},
		{
			name:  "pattern: api_key=value",
			input: "api_key=key123 data",
			want:  "api_key=[REDACTED] data",
		},
		{
			name:  "no match",
			input: "Hello world",
			want:  "Hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactSecrets(tt.input)
			if got != tt.want {
				t.Errorf("RedactSecrets(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRedactWriter(t *testing.T) {
	loadedToken = "mysecret"
	defer func() { loadedToken = "" }()

	var buf bytes.Buffer
	w := NewRedactWriter(&buf)

	_, err := w.Write([]byte("the value mysecret should be hidden"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	got := buf.String()
	want := "the value [REDACTED] should be hidden"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
