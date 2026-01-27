// Package secrets provides secure credential storage using OS keyring or encrypted file backend.
package secrets

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sanisideup/jira-cli/pkg/config"
)

// Backend represents the type of secret storage backend
type Backend string

const (
	// BackendAuto automatically selects the best backend for the platform
	BackendAuto Backend = "auto"
	// BackendKeychain uses the OS keyring (macOS Keychain, Windows Credential Manager, etc.)
	BackendKeychain Backend = "keychain"
	// BackendFile uses an encrypted file-based storage
	BackendFile Backend = "file"

	// Service name for keyring
	ServiceName = "jira-cli"
	// EncryptedFileName is the name of the encrypted credentials file
	EncryptedFileName = "credentials.enc"
)

// Store handles secure credential storage
type Store struct {
	backend Backend
}

// Credentials holds the sensitive data to be stored
type Credentials struct {
	APIToken string `json:"api_token"`
}

// ErrKeyringUnavailable is returned when the keyring is not available
var ErrKeyringUnavailable = errors.New("keyring unavailable on this platform")

// ErrPasswordRequired is returned when password is required for file backend
var ErrPasswordRequired = errors.New("password required for file backend (set JIRA_KEYRING_PASSWORD)")

// NewStore creates a new secret store with the specified backend
func NewStore(backend Backend) *Store {
	if backend == BackendAuto {
		backend = selectBestBackend()
	}
	return &Store{backend: backend}
}

// selectBestBackend chooses the most appropriate backend for the current platform
func selectBestBackend() Backend {
	// Check if running in CI or headless environment
	if os.Getenv("CI") != "" || os.Getenv("JIRA_KEYRING_BACKEND") == "file" {
		return BackendFile
	}

	// Check for keyring availability based on platform
	switch runtime.GOOS {
	case "darwin":
		// macOS has Keychain
		return BackendKeychain
	case "windows":
		// Windows has Credential Manager
		return BackendKeychain
	case "linux":
		// Linux may have keyring (GNOME Keyring, KWallet) but it's not guaranteed
		// Check if DISPLAY or WAYLAND_DISPLAY is set (GUI available)
		if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
			return BackendKeychain
		}
		return BackendFile
	default:
		return BackendFile
	}
}

// GetBackend returns the current backend type
func (s *Store) GetBackend() Backend {
	return s.backend
}

// Store saves credentials securely
func (s *Store) Store(account string, creds *Credentials) error {
	switch s.backend {
	case BackendKeychain:
		return s.storeKeychain(account, creds)
	case BackendFile:
		return s.storeFile(account, creds)
	default:
		return fmt.Errorf("unknown backend: %s", s.backend)
	}
}

// Retrieve loads credentials from secure storage
func (s *Store) Retrieve(account string) (*Credentials, error) {
	switch s.backend {
	case BackendKeychain:
		return s.retrieveKeychain(account)
	case BackendFile:
		return s.retrieveFile(account)
	default:
		return nil, fmt.Errorf("unknown backend: %s", s.backend)
	}
}

// Delete removes credentials from secure storage
func (s *Store) Delete(account string) error {
	switch s.backend {
	case BackendKeychain:
		return s.deleteKeychain(account)
	case BackendFile:
		return s.deleteFile(account)
	default:
		return fmt.Errorf("unknown backend: %s", s.backend)
	}
}

// storeKeychain stores credentials in the OS keyring
func (s *Store) storeKeychain(account string, creds *Credentials) error {
	// Use platform-specific keyring implementation
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	return keyringSet(ServiceName, account, string(data))
}

// retrieveKeychain retrieves credentials from the OS keyring
func (s *Store) retrieveKeychain(account string) (*Credentials, error) {
	data, err := keyringGet(ServiceName, account)
	if err != nil {
		return nil, err
	}

	var creds Credentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		// Fallback: treat as raw API token (for backward compatibility)
		creds.APIToken = data
	}

	return &creds, nil
}

// deleteKeychain removes credentials from the OS keyring
func (s *Store) deleteKeychain(account string) error {
	return keyringDelete(ServiceName, account)
}

// storeFile stores credentials in an encrypted file
func (s *Store) storeFile(account string, creds *Credentials) error {
	password := os.Getenv("JIRA_KEYRING_PASSWORD")
	if password == "" {
		return ErrPasswordRequired
	}

	// Get credentials file path
	filePath, err := getCredentialsFilePath()
	if err != nil {
		return err
	}

	// Load existing credentials or create new map
	allCreds := make(map[string]*Credentials)
	if data, err := os.ReadFile(filePath); err == nil {
		decrypted, err := decrypt(data, password)
		if err == nil {
			json.Unmarshal(decrypted, &allCreds)
		}
	}

	// Add/update account credentials
	allCreds[account] = creds

	// Serialize and encrypt
	data, err := json.Marshal(allCreds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	encrypted, err := encrypt(data, password)
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	// Ensure config directory exists
	configDir, err := config.GetConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, config.ConfigDirPerms); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write encrypted file
	if err := os.WriteFile(filePath, encrypted, config.ConfigFilePerms); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// retrieveFile retrieves credentials from encrypted file
func (s *Store) retrieveFile(account string) (*Credentials, error) {
	password := os.Getenv("JIRA_KEYRING_PASSWORD")
	if password == "" {
		return nil, ErrPasswordRequired
	}

	filePath, err := getCredentialsFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no credentials found for account %s", account)
		}
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	decrypted, err := decrypt(data, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	var allCreds map[string]*Credentials
	if err := json.Unmarshal(decrypted, &allCreds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	creds, ok := allCreds[account]
	if !ok {
		return nil, fmt.Errorf("no credentials found for account %s", account)
	}

	return creds, nil
}

// deleteFile removes credentials from encrypted file
func (s *Store) deleteFile(account string) error {
	password := os.Getenv("JIRA_KEYRING_PASSWORD")
	if password == "" {
		return ErrPasswordRequired
	}

	filePath, err := getCredentialsFilePath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to read credentials file: %w", err)
	}

	decrypted, err := decrypt(data, password)
	if err != nil {
		return fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	var allCreds map[string]*Credentials
	if err := json.Unmarshal(decrypted, &allCreds); err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	delete(allCreds, account)

	// Re-encrypt and save
	newData, err := json.Marshal(allCreds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	encrypted, err := encrypt(newData, password)
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	return os.WriteFile(filePath, encrypted, config.ConfigFilePerms)
}

// getCredentialsFilePath returns the path to the encrypted credentials file
func getCredentialsFilePath() (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, EncryptedFileName), nil
}

// Simple XOR-based encryption with base64 encoding
// Note: For production, consider using a proper encryption library like golang.org/x/crypto/nacl/secretbox
func encrypt(data []byte, password string) ([]byte, error) {
	key := []byte(password)
	encrypted := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		encrypted[i] = data[i] ^ key[i%len(key)]
	}
	encoded := base64.StdEncoding.EncodeToString(encrypted)
	return []byte(encoded), nil
}

func decrypt(data []byte, password string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	key := []byte(password)
	decrypted := make([]byte, len(decoded))
	for i := 0; i < len(decoded); i++ {
		decrypted[i] = decoded[i] ^ key[i%len(key)]
	}
	return decrypted, nil
}
