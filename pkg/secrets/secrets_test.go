package secrets

import (
	"os"
	"runtime"
	"testing"
)

func TestNewStore_AutoBackend(t *testing.T) {
	// Save and restore environment
	origCI := os.Getenv("CI")
	origBackend := os.Getenv("JIRA_KEYRING_BACKEND")
	defer func() {
		os.Setenv("CI", origCI)
		os.Setenv("JIRA_KEYRING_BACKEND", origBackend)
	}()

	tests := []struct {
		name         string
		ci           string
		backendEnv   string
		wantBackend  Backend
		skipOnLinux  bool
	}{
		{
			name:        "CI environment uses file backend",
			ci:          "true",
			wantBackend: BackendFile,
		},
		{
			name:        "JIRA_KEYRING_BACKEND=file uses file backend",
			backendEnv:  "file",
			wantBackend: BackendFile,
		},
		{
			name:         "macOS/Windows uses keychain by default",
			ci:           "",
			backendEnv:   "",
			wantBackend:  BackendKeychain,
			skipOnLinux:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnLinux && runtime.GOOS == "linux" {
				t.Skip("Skipping on Linux - keychain availability depends on desktop environment")
			}

			os.Setenv("CI", tt.ci)
			os.Setenv("JIRA_KEYRING_BACKEND", tt.backendEnv)

			store := NewStore(BackendAuto)
			if store.GetBackend() != tt.wantBackend {
				t.Errorf("NewStore(BackendAuto) backend = %v, want %v", store.GetBackend(), tt.wantBackend)
			}
		})
	}
}

func TestStore_Keychain_StoreAndRetrieve(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("Skipping keychain test on non-macOS/Windows platform")
	}

	store := NewStore(BackendKeychain)
	account := "test-user@example.com"
	creds := &Credentials{APIToken: "test-api-token-12345"}

	// Store credentials
	if err := store.Store(account, creds); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Retrieve credentials
	retrieved, err := store.Retrieve(account)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if retrieved.APIToken != creds.APIToken {
		t.Errorf("Retrieve() APIToken = %v, want %v", retrieved.APIToken, creds.APIToken)
	}

	// Cleanup
	if err := store.Delete(account); err != nil {
		t.Errorf("Delete() error = %v", err)
	}
}

func TestStore_Keychain_Delete(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("Skipping keychain test on non-macOS/Windows platform")
	}

	store := NewStore(BackendKeychain)
	account := "test-delete@example.com"
	creds := &Credentials{APIToken: "test-token-to-delete"}

	// Store credentials
	if err := store.Store(account, creds); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Delete credentials
	if err := store.Delete(account); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion - retrieve should fail
	_, err := store.Retrieve(account)
	if err == nil {
		t.Error("Retrieve() after Delete() should fail, but succeeded")
	}
}

func TestStore_File_StoreAndRetrieve(t *testing.T) {
	// Set up test password
	origPassword := os.Getenv("JIRA_KEYRING_PASSWORD")
	os.Setenv("JIRA_KEYRING_PASSWORD", "test-password-123")
	defer os.Setenv("JIRA_KEYRING_PASSWORD", origPassword)

	store := NewStore(BackendFile)
	account := "test-file-user@example.com"
	creds := &Credentials{APIToken: "test-file-api-token-67890"}

	// Store credentials
	if err := store.Store(account, creds); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Retrieve credentials
	retrieved, err := store.Retrieve(account)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if retrieved.APIToken != creds.APIToken {
		t.Errorf("Retrieve() APIToken = %v, want %v", retrieved.APIToken, creds.APIToken)
	}

	// Cleanup
	if err := store.Delete(account); err != nil {
		t.Errorf("Delete() error = %v", err)
	}
}

func TestStore_File_Delete(t *testing.T) {
	// Set up test password
	origPassword := os.Getenv("JIRA_KEYRING_PASSWORD")
	os.Setenv("JIRA_KEYRING_PASSWORD", "test-password-123")
	defer os.Setenv("JIRA_KEYRING_PASSWORD", origPassword)

	store := NewStore(BackendFile)
	account := "test-file-delete@example.com"
	creds := &Credentials{APIToken: "test-token-to-delete"}

	// Store credentials
	if err := store.Store(account, creds); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Delete credentials
	if err := store.Delete(account); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion - retrieve should fail
	_, err := store.Retrieve(account)
	if err == nil {
		t.Error("Retrieve() after Delete() should fail, but succeeded")
	}
}

func TestStore_File_RequiresPassword(t *testing.T) {
	// Ensure password is not set
	origPassword := os.Getenv("JIRA_KEYRING_PASSWORD")
	os.Unsetenv("JIRA_KEYRING_PASSWORD")
	defer func() {
		if origPassword != "" {
			os.Setenv("JIRA_KEYRING_PASSWORD", origPassword)
		}
	}()

	store := NewStore(BackendFile)
	account := "test-no-password@example.com"
	creds := &Credentials{APIToken: "test-token"}

	// Store should fail without password
	err := store.Store(account, creds)
	if err != ErrPasswordRequired {
		t.Errorf("Store() without password error = %v, want %v", err, ErrPasswordRequired)
	}

	// Retrieve should fail without password
	_, err = store.Retrieve(account)
	if err != ErrPasswordRequired {
		t.Errorf("Retrieve() without password error = %v, want %v", err, ErrPasswordRequired)
	}
}

func TestStore_File_MultipleAccounts(t *testing.T) {
	// Set up test password
	origPassword := os.Getenv("JIRA_KEYRING_PASSWORD")
	os.Setenv("JIRA_KEYRING_PASSWORD", "test-password-456")
	defer os.Setenv("JIRA_KEYRING_PASSWORD", origPassword)

	store := NewStore(BackendFile)

	accounts := map[string]*Credentials{
		"user1@example.com": {APIToken: "token-1"},
		"user2@example.com": {APIToken: "token-2"},
		"user3@example.com": {APIToken: "token-3"},
	}

	// Store all accounts
	for account, creds := range accounts {
		if err := store.Store(account, creds); err != nil {
			t.Fatalf("Store(%s) error = %v", account, err)
		}
	}

	// Verify all accounts can be retrieved
	for account, expectedCreds := range accounts {
		retrieved, err := store.Retrieve(account)
		if err != nil {
			t.Fatalf("Retrieve(%s) error = %v", account, err)
		}
		if retrieved.APIToken != expectedCreds.APIToken {
			t.Errorf("Retrieve(%s) APIToken = %v, want %v", account, retrieved.APIToken, expectedCreds.APIToken)
		}
	}

	// Delete one account and verify others still work
	delete(accounts, "user2@example.com")
	if err := store.Delete("user2@example.com"); err != nil {
		t.Fatalf("Delete(user2@example.com) error = %v", err)
	}

	// user2 should now fail
	_, err := store.Retrieve("user2@example.com")
	if err == nil {
		t.Error("Retrieve(user2@example.com) after Delete() should fail")
	}

	// Others should still work
	for account, expectedCreds := range accounts {
		retrieved, err := store.Retrieve(account)
		if err != nil {
			t.Fatalf("Retrieve(%s) after delete error = %v", account, err)
		}
		if retrieved.APIToken != expectedCreds.APIToken {
			t.Errorf("Retrieve(%s) APIToken = %v, want %v", account, retrieved.APIToken, expectedCreds.APIToken)
		}
	}

	// Cleanup remaining accounts
	for account := range accounts {
		store.Delete(account)
	}
}
