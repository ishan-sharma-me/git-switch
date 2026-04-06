package gpg

import (
	"fmt"
	"os/exec"
	"strings"
)

// GPGKey represents a GPG secret key.
type GPGKey struct {
	KeyID  string // Long key ID
	UserID string // "Name <email>"
}

// ListSecretKeys returns all GPG secret keys on the system.
func ListSecretKeys() ([]GPGKey, error) {
	cmd := exec.Command("gpg", "--list-secret-keys", "--keyid-format", "LONG", "--with-colons")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gpg not available: %w", err)
	}

	var keys []GPGKey
	var currentKeyID string

	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Split(line, ":")

		// sec line has the key ID
		if len(fields) >= 5 && fields[0] == "sec" {
			currentKeyID = fields[4]
		}

		// uid line has the user ID
		if len(fields) >= 10 && fields[0] == "uid" && currentKeyID != "" {
			keys = append(keys, GPGKey{
				KeyID:  currentKeyID,
				UserID: fields[9],
			})
			currentKeyID = ""
		}
	}

	return keys, nil
}

// ValidateKey checks that a GPG secret key exists and can sign.
func ValidateKey(keyID string) error {
	// Check key exists
	cmd := exec.Command("gpg", "--list-secret-keys", "--keyid-format", "LONG", keyID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("GPG key %s not found", keyID)
	}

	// Test signing
	cmd = exec.Command("gpg", "--batch", "--yes", "--clear-sign", "-u", keyID)
	cmd.Stdin = strings.NewReader("git-switch test")
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if strings.Contains(text, "No secret key") {
			return fmt.Errorf("GPG key %s: no secret key", keyID)
		}
		if strings.Contains(text, "expired") {
			return fmt.Errorf("GPG key %s: expired", keyID)
		}
		return fmt.Errorf("GPG key %s: cannot sign: %s", keyID, text)
	}
	return nil
}

// GenerateKey creates a new GPG key (RSA 4096) in batch mode.
// Returns the key ID.
func GenerateKey(name, email string) (string, error) {
	batchConfig := fmt.Sprintf(`%%no-protection
Key-Type: RSA
Key-Length: 4096
Subkey-Type: RSA
Subkey-Length: 4096
Name-Real: %s
Name-Email: %s
Expire-Date: 0
%%commit
`, name, email)

	cmd := exec.Command("gpg", "--batch", "--gen-key")
	cmd.Stdin = strings.NewReader(batchConfig)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gpg key generation failed: %s", strings.TrimSpace(string(output)))
	}

	// Find the newly created key by email
	keys, err := ListSecretKeys()
	if err != nil {
		return "", err
	}
	for _, k := range keys {
		if strings.Contains(k.UserID, email) {
			return k.KeyID, nil
		}
	}
	return "", fmt.Errorf("key generated but could not find it")
}

// ExportPublicKey exports the GPG public key in ASCII armor format.
func ExportPublicKey(keyID string) (string, error) {
	cmd := exec.Command("gpg", "--armor", "--export", keyID)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("exporting GPG key: %w", err)
	}
	return string(output), nil
}
