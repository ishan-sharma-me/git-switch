package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// SetGlobalUser sets git global user.name and user.email.
func SetGlobalUser(name, email string) error {
	if err := gitConfig("user.name", name); err != nil {
		return fmt.Errorf("setting user.name: %w", err)
	}
	if err := gitConfig("user.email", email); err != nil {
		return fmt.Errorf("setting user.email: %w", err)
	}
	return nil
}

// SetGlobalGPG enables GPG commit signing with the given key ID.
func SetGlobalGPG(gpgKey string) error {
	if err := gitConfig("user.signingkey", gpgKey); err != nil {
		return fmt.Errorf("setting signingkey: %w", err)
	}
	if err := gitConfig("commit.gpgsign", "true"); err != nil {
		return fmt.Errorf("enabling gpgsign: %w", err)
	}
	return nil
}

// DisableGPG disables GPG commit signing.
func DisableGPG() error {
	// Unset signingkey (ignore error if not set)
	exec.Command("git", "config", "--global", "--unset", "user.signingkey").Run()
	if err := gitConfig("commit.gpgsign", "false"); err != nil {
		return fmt.Errorf("disabling gpgsign: %w", err)
	}
	return nil
}

// SetLocalUser sets git local (repo-level) user.name and user.email.
func SetLocalUser(name, email string) error {
	if err := gitLocalConfig("user.name", name); err != nil {
		return fmt.Errorf("setting user.name: %w", err)
	}
	if err := gitLocalConfig("user.email", email); err != nil {
		return fmt.Errorf("setting user.email: %w", err)
	}
	return nil
}

// SetLocalGPG enables GPG commit signing at the repo level with the given key ID.
func SetLocalGPG(gpgKey string) error {
	if err := gitLocalConfig("user.signingkey", gpgKey); err != nil {
		return fmt.Errorf("setting signingkey: %w", err)
	}
	if err := gitLocalConfig("commit.gpgsign", "true"); err != nil {
		return fmt.Errorf("enabling gpgsign: %w", err)
	}
	return nil
}

// DisableLocalGPG disables GPG commit signing at the repo level.
func DisableLocalGPG() error {
	exec.Command("git", "config", "--local", "--unset", "user.signingkey").Run()
	if err := gitLocalConfig("commit.gpgsign", "false"); err != nil {
		return fmt.Errorf("disabling gpgsign: %w", err)
	}
	return nil
}

// GetGlobalUser returns the current global user.name and user.email.
func GetGlobalUser() (name, email string, err error) {
	name, err = gitConfigGet("user.name")
	if err != nil {
		return "", "", err
	}
	email, err = gitConfigGet("user.email")
	if err != nil {
		return "", "", err
	}
	return name, email, nil
}

// GetGlobalSigningKey returns the current global user.signingkey, or empty string.
func GetGlobalSigningKey() string {
	key, _ := gitConfigGet("user.signingkey")
	return key
}

func gitConfig(key, value string) error {
	cmd := exec.Command("git", "config", "--global", key, value)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", key, strings.TrimSpace(string(output)))
	}
	return nil
}

func gitLocalConfig(key, value string) error {
	cmd := exec.Command("git", "config", "--local", key, value)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", key, strings.TrimSpace(string(output)))
	}
	return nil
}

// IsInsideWorkTree returns true if the current directory is inside a git repo.
func IsInsideWorkTree() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

func gitConfigGet(key string) (string, error) {
	cmd := exec.Command("git", "config", "--global", "--get", key)
	output, err := cmd.Output()
	if err != nil {
		return "", nil // key not set is not an error
	}
	return strings.TrimSpace(string(output)), nil
}
