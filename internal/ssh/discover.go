package ssh

import (
	"os"
	"path/filepath"
	"strings"
)

// DiscoveredKey represents an SSH key found on disk.
type DiscoveredKey struct {
	Path        string // Absolute path to private key
	Fingerprint string
	Comment     string
}

// DiscoverKeys finds SSH private keys in ~/.ssh/ that have matching .pub files.
// Excludes infrastructure keys (.pem), config files, and known_hosts.
func DiscoverKeys() ([]DiscoveredKey, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	sshDir := filepath.Join(home, ".ssh")

	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil, err
	}

	skipNames := map[string]bool{
		"config":          true,
		"known_hosts":     true,
		"known_hosts.old": true,
		"authorized_keys": true,
		"agent":           true,
	}

	var keys []DiscoveredKey
	for _, e := range entries {
		name := e.Name()

		// Skip directories
		if e.IsDir() {
			continue
		}
		// Skip hidden files
		if strings.HasPrefix(name, ".") {
			continue
		}
		// Skip known non-key files
		if skipNames[name] {
			continue
		}
		// Skip .pub, .pem, .crt.pem files
		if strings.HasSuffix(name, ".pub") ||
			strings.HasSuffix(name, ".pem") ||
			strings.HasSuffix(name, ".crt") {
			continue
		}
		// Skip backup files
		if strings.Contains(name, "backup") || strings.Contains(name, "tmp") {
			continue
		}

		privPath := filepath.Join(sshDir, name)

		// Must have a matching .pub file
		pubPath := privPath + ".pub"
		if _, err := os.Stat(pubPath); os.IsNotExist(err) {
			continue
		}

		fingerprint, comment, err := GetKeyFingerprint(privPath)
		if err != nil {
			continue // skip unreadable keys
		}

		keys = append(keys, DiscoveredKey{
			Path:        privPath,
			Fingerprint: fingerprint,
			Comment:     comment,
		})
	}

	return keys, nil
}
