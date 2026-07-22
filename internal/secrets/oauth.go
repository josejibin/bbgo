package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zalando/go-keyring"
)

const oauthAccountName = "bitbucket-oauth"

// OAuthCredentials is the persisted OAuth session: consumer credentials plus
// the current token set. Stored as JSON in the OS keychain, falling back to
// an AES-GCM encrypted file (~/.bbgo/oauth).
type OAuthCredentials struct {
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Expired reports whether the access token is expired or about to expire.
func (o *OAuthCredentials) Expired() bool {
	return time.Now().After(o.ExpiresAt.Add(-60 * time.Second))
}

// StoreOAuth persists the OAuth session and registers its secrets for redaction.
func StoreOAuth(creds *OAuthCredentials) error {
	registerOAuthSecrets(creds)
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshaling OAuth credentials: %w", err)
	}
	if err := keyring.Set(serviceName, oauthAccountName, string(data)); err == nil {
		removeOAuthFallbackFile()
		return nil
	}
	path, err := oauthFallbackPath()
	if err != nil {
		return err
	}
	return encryptToFile(path, data)
}

// LoadOAuth retrieves the OAuth session. Returns (nil, nil) when no session
// is stored — that is the "not logged in" case, not an error.
func LoadOAuth() (*OAuthCredentials, error) {
	raw, keyringErr := keyring.Get(serviceName, oauthAccountName)
	var data []byte
	if keyringErr == nil && raw != "" {
		data = []byte(raw)
	} else {
		path, err := oauthFallbackPath()
		if err != nil {
			return nil, err
		}
		data, err = decryptFromFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
	}

	var creds OAuthCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parsing stored OAuth credentials: %w", err)
	}
	registerOAuthSecrets(&creds)
	return &creds, nil
}

// ClearOAuth removes the OAuth session from keychain and fallback file.
func ClearOAuth() error {
	_ = keyring.Delete(serviceName, oauthAccountName)
	removeOAuthFallbackFile()
	return nil
}

func registerOAuthSecrets(creds *OAuthCredentials) {
	registerSecret(creds.AccessToken)
	registerSecret(creds.RefreshToken)
	registerSecret(creds.ClientSecret)
}

func oauthFallbackPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".bbgo", "oauth"), nil
}

func removeOAuthFallbackFile() {
	path, err := oauthFallbackPath()
	if err != nil {
		return
	}
	os.Remove(path)
}
