package secrets

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// keyringSet stores a secret in the OS keyring
func keyringSet(service, account, secret string) error {
	switch runtime.GOOS {
	case "darwin":
		return macOSKeyringSet(service, account, secret)
	case "windows":
		return windowsCredentialSet(service, account, secret)
	case "linux":
		return linuxSecretServiceSet(service, account, secret)
	default:
		return ErrKeyringUnavailable
	}
}

// keyringGet retrieves a secret from the OS keyring
func keyringGet(service, account string) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return macOSKeyringGet(service, account)
	case "windows":
		return windowsCredentialGet(service, account)
	case "linux":
		return linuxSecretServiceGet(service, account)
	default:
		return "", ErrKeyringUnavailable
	}
}

// keyringDelete removes a secret from the OS keyring
func keyringDelete(service, account string) error {
	switch runtime.GOOS {
	case "darwin":
		return macOSKeyringDelete(service, account)
	case "windows":
		return windowsCredentialDelete(service, account)
	case "linux":
		return linuxSecretServiceDelete(service, account)
	default:
		return ErrKeyringUnavailable
	}
}

// macOS Keychain implementation using `security` command
func macOSKeyringSet(service, account, secret string) error {
	// First try to delete any existing entry
	_ = macOSKeyringDelete(service, account)

	cmd := exec.Command("security", "add-generic-password",
		"-s", service,
		"-a", account,
		"-w", secret,
		"-U", // Update if exists
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store in keychain: %w", err)
	}
	return nil
}

func macOSKeyringGet(service, account string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", service,
		"-a", account,
		"-w", // Output only the password
	)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve from keychain: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func macOSKeyringDelete(service, account string) error {
	cmd := exec.Command("security", "delete-generic-password",
		"-s", service,
		"-a", account,
	)
	// Ignore errors (entry might not exist)
	_ = cmd.Run()
	return nil
}

// Windows Credential Manager implementation using PowerShell
func windowsCredentialSet(service, account, secret string) error {
	target := fmt.Sprintf("%s:%s", service, account)
	psScript := fmt.Sprintf(`
		$cred = New-Object System.Management.Automation.PSCredential('%s', (ConvertTo-SecureString '%s' -AsPlainText -Force))
		cmdkey /generic:%s /user:%s /pass:%s
	`, account, secret, target, account, secret)

	cmd := exec.Command("powershell", "-Command", psScript)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store in credential manager: %w", err)
	}
	return nil
}

func windowsCredentialGet(service, account string) (string, error) {
	target := fmt.Sprintf("%s:%s", service, account)
	psScript := fmt.Sprintf(`
		$cred = cmdkey /list:%s 2>&1
		if ($cred -match 'Password:(.+)') { $matches[1].Trim() }
	`, target)

	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve from credential manager: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func windowsCredentialDelete(service, account string) error {
	target := fmt.Sprintf("%s:%s", service, account)
	cmd := exec.Command("cmdkey", "/delete:"+target)
	_ = cmd.Run()
	return nil
}

// Linux Secret Service (GNOME Keyring / KWallet) implementation using `secret-tool`
func linuxSecretServiceSet(service, account, secret string) error {
	cmd := exec.Command("secret-tool", "store",
		"--label", fmt.Sprintf("%s - %s", service, account),
		"service", service,
		"account", account,
	)
	cmd.Stdin = strings.NewReader(secret)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store in secret service: %w", err)
	}
	return nil
}

func linuxSecretServiceGet(service, account string) (string, error) {
	cmd := exec.Command("secret-tool", "lookup",
		"service", service,
		"account", account,
	)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve from secret service: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func linuxSecretServiceDelete(service, account string) error {
	cmd := exec.Command("secret-tool", "clear",
		"service", service,
		"account", account,
	)
	_ = cmd.Run()
	return nil
}
