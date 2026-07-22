package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEncryptDecryptFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	plaintext := []byte(`{"access_token":"at-1"}`)

	if err := encryptToFile(path, plaintext); err != nil {
		t.Fatalf("encryptToFile: %v", err)
	}

	// Ciphertext must not contain the plaintext
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if strings.Contains(string(raw), "at-1") {
		t.Error("token stored in plaintext")
	}

	got, err := decryptFromFile(path)
	if err != nil {
		t.Fatalf("decryptFromFile: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Errorf("round trip = %q, want %q", got, plaintext)
	}
}

func TestDecryptMissingFileReturnsNotExist(t *testing.T) {
	_, err := decryptFromFile(filepath.Join(t.TempDir(), "nope"))
	if !os.IsNotExist(err) {
		t.Errorf("expected IsNotExist error, got %v", err)
	}
}

func TestDecryptTamperedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := encryptToFile(path, []byte("data")); err != nil {
		t.Fatalf("encryptToFile: %v", err)
	}
	raw, _ := os.ReadFile(path)
	raw[len(raw)-1] ^= 0xFF
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatalf("writing tampered file: %v", err)
	}
	if _, err := decryptFromFile(path); err == nil {
		t.Error("expected error for tampered ciphertext")
	}
}

func TestOAuthCredentialsExpired(t *testing.T) {
	fresh := &OAuthCredentials{ExpiresAt: time.Now().Add(1 * time.Hour)}
	if fresh.Expired() {
		t.Error("token expiring in 1h reported as expired")
	}
	soon := &OAuthCredentials{ExpiresAt: time.Now().Add(30 * time.Second)}
	if !soon.Expired() {
		t.Error("token expiring in 30s should count as expired (60s leeway)")
	}
	past := &OAuthCredentials{ExpiresAt: time.Now().Add(-1 * time.Minute)}
	if !past.Expired() {
		t.Error("past-expiry token reported as valid")
	}
}

func TestRegisteredSecretsRedacted(t *testing.T) {
	registerOAuthSecrets(&OAuthCredentials{
		AccessToken:  "oauth-access-xyz",
		RefreshToken: "oauth-refresh-xyz",
		ClientSecret: "consumer-secret-xyz",
	})
	out := RedactSecrets("a oauth-access-xyz b oauth-refresh-xyz c consumer-secret-xyz d")
	for _, leaked := range []string{"oauth-access-xyz", "oauth-refresh-xyz", "consumer-secret-xyz"} {
		if strings.Contains(out, leaked) {
			t.Errorf("output leaked %q: %s", leaked, out)
		}
	}
	if !strings.Contains(out, "a [REDACTED] b [REDACTED] c [REDACTED] d") {
		t.Errorf("unexpected redacted output: %s", out)
	}
}
