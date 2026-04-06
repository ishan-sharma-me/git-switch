package ssh

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// EnsureGitHubKnownHost checks that github.com is in ~/.ssh/known_hosts.
// If missing, fetches GitHub's host keys via ssh-keyscan and appends them.
func EnsureGitHubKnownHost() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	sshDir := filepath.Join(home, ".ssh")
	knownHostsPath := filepath.Join(sshDir, "known_hosts")

	// Ensure ~/.ssh directory exists
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return fmt.Errorf("creating ~/.ssh: %w", err)
	}

	// Check if github.com already present
	if hasGitHubEntry(knownHostsPath) {
		return nil
	}

	// Fetch keys via ssh-keyscan
	cmd := exec.Command("ssh-keyscan", "-t", "ed25519,rsa,ecdsa", "github.com")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ssh-keyscan failed: %w", err)
	}

	// Append to known_hosts
	f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening known_hosts: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(output); err != nil {
		return fmt.Errorf("writing known_hosts: %w", err)
	}

	return nil
}

func hasGitHubEntry(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "github.com ") || strings.HasPrefix(line, "github.com,") {
			return true
		}
	}
	return false
}
