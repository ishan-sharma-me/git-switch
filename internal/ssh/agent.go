package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// AddKeyToAgent adds the SSH private key to the ssh-agent.
// On macOS, uses --apple-use-keychain for Keychain integration.
func AddKeyToAgent(keyPath string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("ssh-add", "--apple-use-keychain", keyPath)
	} else {
		cmd = exec.Command("ssh-add", keyPath)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-add failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// TestGitHubAuth tests SSH authentication to GitHub using a specific key.
// Returns the GitHub username on success.
// Note: GitHub returns exit code 1 even on successful auth.
func TestGitHubAuth(keyPath string) (string, error) {
	cmd := exec.Command("ssh",
		"-i", keyPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "BatchMode=yes",
		"-T", "git@github.com",
	)
	output, _ := cmd.CombinedOutput()
	text := string(output)

	// Parse "Hi <username>! You've successfully authenticated..."
	re := regexp.MustCompile(`Hi ([^!]+)!`)
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 2 {
		return matches[1], nil
	}

	return "", fmt.Errorf("authentication failed: %s", strings.TrimSpace(text))
}

// GenerateKey creates a new ed25519 SSH key pair.
func GenerateKey(path, comment string) error {
	cmd := exec.Command("ssh-keygen",
		"-t", "ed25519",
		"-C", comment,
		"-f", path,
		"-N", "", // empty passphrase
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-keygen failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// GetKeyFingerprint returns the fingerprint and comment of an SSH key.
func GetKeyFingerprint(keyPath string) (fingerprint, comment string, err error) {
	cmd := exec.Command("ssh-keygen", "-lf", keyPath)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("cannot read key: %w", err)
	}
	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) >= 2 {
		fingerprint = parts[1]
	}
	if len(parts) >= 3 {
		comment = parts[2]
	}
	return fingerprint, comment, nil
}

// ReadPublicKey reads and returns the contents of a .pub key file.
func ReadPublicKey(privateKeyPath string) (string, error) {
	pubPath := privateKeyPath + ".pub"
	data, err := os.ReadFile(pubPath)
	if err != nil {
		return "", fmt.Errorf("reading public key: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}
