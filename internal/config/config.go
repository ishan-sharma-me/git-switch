package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	configDir  = ".config/git-switch"
	configFile = "config.yaml"
)

// Account represents a single managed Git identity.
type Account struct {
	SSHKey string `yaml:"ssh_key"`           // Path to private key (supports ~ prefix)
	Name   string `yaml:"name"`              // git user.name
	Email  string `yaml:"email"`             // git user.email
	GPGKey string `yaml:"gpg_key,omitempty"` // GPG key ID, empty if none
}

// Config is the top-level git-switch configuration.
type Config struct {
	Active   string              `yaml:"active"`   // Name of currently active account
	Accounts map[string]*Account `yaml:"accounts"` // account-name -> Account
}

// ConfigPath returns the absolute path to the config file.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, configDir, configFile), nil
}

// Load reads the config from disk. Returns a zero-value Config (not error)
// if the file does not exist, so callers can bootstrap.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{Accounts: make(map[string]*Account)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if cfg.Accounts == nil {
		cfg.Accounts = make(map[string]*Account)
	}
	return &cfg, nil
}

// Save writes the config to disk, creating parent directories as needed.
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// AccountNames returns a sorted slice of all account names (for completion).
func (c *Config) AccountNames() []string {
	names := make([]string, 0, len(c.Accounts))
	for name := range c.Accounts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ExpandPath resolves ~ to the home directory in a path.
func ExpandPath(p string) (string, error) {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}

// ShortenPath replaces the home directory prefix with ~.
func ShortenPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}
