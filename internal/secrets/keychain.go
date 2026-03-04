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

// Token returns the currently loaded token.
func Token() string {
	return loadedToken
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

// LoadToken retrieves the token from OS keychain or fallback file.
func LoadToken() (string, error) {
	token, err := keyring.Get(serviceName, accountName)
	if err == nil && token != "" {
		loadedToken = token
		return token, nil
	}
	// Fallback to encrypted file
	token, err = loadTokenFile()
	if err != nil {
		return "", err
	}
	loadedToken = token
	return token, nil
}

// ClearToken removes the token from keychain and fallback file.
func ClearToken() error {
	loadedToken = ""
	_ = keyring.Delete(serviceName, accountName)
	removeFallbackFile()
	return nil
}

// --- Encrypted file fallback ---

func fallbackPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".bbgo", "token")
}

func deriveKey() []byte {
	// Derive a key from hostname + username (not secret, but ties token to machine/user)
	hostname, _ := os.Hostname()
	user := os.Getenv("USER")
	if runtime.GOOS == "windows" {
		user = os.Getenv("USERNAME")
	}
	h := sha256.Sum256([]byte("bbgo:" + hostname + ":" + user))
	return h[:]
}

func storeTokenFile(token string) error {
	key := deriveKey()
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

	path := fallbackPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating dir: %w", err)
	}
	return os.WriteFile(path, ciphertext, 0600)
}

func loadTokenFile() (string, error) {
	path := fallbackPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no token stored — run `bbgo config set --token`")
		}
		return "", fmt.Errorf("reading token file: %w", err)
	}

	key := deriveKey()
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
	os.Remove(fallbackPath())
}
