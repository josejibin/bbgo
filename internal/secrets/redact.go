package secrets

import (
	"io"
	"regexp"
	"slices"
	"strings"
)

var secretPattern = regexp.MustCompile(
	`(?i)(token|password|secret|api[_-]?key)\s*[:=]\s*\S+`,
)

// registeredSecrets holds additional literal secrets (OAuth tokens, client
// secrets) that must never appear in output.
var registeredSecrets []string

// RegisterSecret adds a literal value to the redaction list.
func RegisterSecret(s string) { registerSecret(s) }

// registerSecret adds a literal value to the redaction list.
func registerSecret(s string) {
	if s == "" || slices.Contains(registeredSecrets, s) {
		return
	}
	registeredSecrets = append(registeredSecrets, s)
}

// RedactSecrets replaces the loaded token, registered secrets, and known
// secret patterns in s.
func RedactSecrets(s string) string {
	if loadedToken != "" {
		s = strings.ReplaceAll(s, loadedToken, "[REDACTED]")
	}
	for _, secret := range registeredSecrets {
		s = strings.ReplaceAll(s, secret, "[REDACTED]")
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
