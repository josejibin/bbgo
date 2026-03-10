package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/zalando/go-keyring"
)

const (
	serviceName = "bbgo"
	accountName = "bitbucket-token"
)

// loadedToken holds the token in memory — unexported, never returned in output.
var loadedToken string

// lastLoadErr stores the most recent LoadToken error for diagnostic messages.
var lastLoadErr error

// Token returns the currently loaded token.
func Token() string {
	return loadedToken
}

// LastLoadError returns the error from the most recent LoadToken call, if any.
// Useful for providing diagnostic messages when Token() returns empty.
func LastLoadError() error {
	return lastLoadErr
}

// StoreToken saves the token to the OS keychain, falling back to encrypted file.
func StoreToken(token string) error {
	loadedToken = token
	err := keyring.Set(serviceName, accountName, token)
	if err == nil {
		// Remove fallback file if keychain works
		removeFallbackFile()
		return nil
	}
	// Fallback to encrypted file
	return storeTokenFile(token)
}

// LoadToken retrieves the token from: BBGO_TOKEN env → OS keychain → fallback file.
func LoadToken() (string, error) {
	// 1. Environment variable (useful for headless/CI environments)
	if token := os.Getenv("BBGO_TOKEN"); token != "" {
		loadedToken = token
		return token, nil
	}

	// 2. OS keychain
	token, keyringErr := keyring.Get(serviceName, accountName)
	if keyringErr == nil && token != "" {
		loadedToken = token
		return token, nil
	}

	// 3. Encrypted file fallback
	token, fileErr := loadTokenFile()
	if fileErr == nil {
		loadedToken = token
		return token, nil
	}

	// All methods failed — provide helpful diagnostics
	if keyringErr != nil {
		lastLoadErr = fmt.Errorf("keychain unavailable (%v), file fallback failed (%v)", keyringErr, fileErr)
		return "", fmt.Errorf("cannot load token: %v — set BBGO_TOKEN env var or run `bbgo config set --token`", lastLoadErr)
	}
	lastLoadErr = fileErr
	return "", fileErr
}

// ClearToken removes the token from keychain and fallback file.
func ClearToken() error {
	loadedToken = ""
	_ = keyring.Delete(serviceName, accountName)
	removeFallbackFile()
	return nil
}

// --- Encrypted file fallback ---

func fallbackPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".bbgo", "token"), nil
}

func deriveKey() ([]byte, error) {
	// Derive a key from hostname + username (not secret, but ties token to machine/user)
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("getting hostname: %w", err)
	}
	user := os.Getenv("USER")
	if runtime.GOOS == "windows" {
		user = os.Getenv("USERNAME")
	}
	h := sha256.Sum256([]byte("bbgo:" + hostname + ":" + user))
	return h[:], nil
}

func storeTokenFile(token string) error {
	key, err := deriveKey()
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("creating GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("generating nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(token), nil)

	path, err := fallbackPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating dir: %w", err)
	}
	return os.WriteFile(path, ciphertext, 0600)
}

func loadTokenFile() (string, error) {
	path, err := fallbackPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no token stored — run `bbgo config set --token`")
		}
		return "", fmt.Errorf("reading token file: %w", err)
	}

	key, err := deriveKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("token file corrupted")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypting token: %w", err)
	}
	return string(plaintext), nil
}

func removeFallbackFile() {
	path, err := fallbackPath()
	if err != nil {
		return
	}
	os.Remove(path)
}
