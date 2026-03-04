package secrets

import (
	"io"
	"regexp"
	"strings"
)

var secretPattern = regexp.MustCompile(
	`(?i)(token|password|secret|api[_-]?key)\s*[:=]\s*\S+`,
)

// RedactSecrets replaces the loaded token and known secret patterns in s.
func RedactSecrets(s string) string {
	if loadedToken != "" {
		s = strings.ReplaceAll(s, loadedToken, "[REDACTED]")
	}
	s = secretPattern.ReplaceAllString(s, "$1=[REDACTED]")
	return s
}

// RedactWriter wraps an io.Writer and redacts secrets from all output.
type RedactWriter struct {
	w io.Writer
}

// NewRedactWriter creates a new RedactWriter wrapping w.
func NewRedactWriter(w io.Writer) *RedactWriter {
	return &RedactWriter{w: w}
}

func (rw *RedactWriter) Write(p []byte) (int, error) {
	redacted := RedactSecrets(string(p))
	n, err := rw.w.Write([]byte(redacted))
	if n != len(redacted) && err == nil {
		err = io.ErrShortWrite
	}
	// Return original length so callers don't see a mismatch
	return len(p), err
}
