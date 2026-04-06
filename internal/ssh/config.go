package ssh

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	markerStart = "# git-switch-managed-start"
	markerEnd   = "# git-switch-managed-end"
)

// UpdateGitHubIdentity sets the SSH IdentityFile for github.com in ~/.ssh/config.
// Uses marker comments to safely manage only its own block.
func UpdateGitHubIdentity(sshKeyPath string) error {
	configPath, err := sshConfigPath()
	if err != nil {
		return err
	}

	before, after, _, err := readSSHConfig(configPath)
	if err != nil {
		return err
	}

	return writeSSHConfig(configPath, before, sshKeyPath, after)
}

// GetCurrentIdentity returns the IdentityFile from the managed block, or empty string.
func GetCurrentIdentity() (string, error) {
	configPath, err := sshConfigPath()
	if err != nil {
		return "", err
	}

	_, _, identityFile, err := readSSHConfig(configPath)
	if err != nil {
		return "", err
	}
	return identityFile, nil
}

func sshConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "config"), nil
}

// readSSHConfig parses ~/.ssh/config and returns:
//   - before: lines before the managed block (or before legacy Host github.com)
//   - after: lines after the managed block (or after legacy Host github.com)
//   - identityFile: current IdentityFile in managed block (empty if none)
func readSSHConfig(path string) (before []string, after []string, identityFile string, err error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil, "", nil
	}
	if err != nil {
		return nil, nil, "", fmt.Errorf("opening SSH config: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, "", fmt.Errorf("reading SSH config: %w", err)
	}

	// Look for managed markers
	startIdx := -1
	endIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == markerStart {
			startIdx = i
		}
		if strings.TrimSpace(line) == markerEnd {
			endIdx = i
		}
	}

	if startIdx >= 0 && endIdx > startIdx {
		// Found managed block — extract identity file
		for _, line := range lines[startIdx : endIdx+1] {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "IdentityFile ") {
				identityFile = strings.TrimSpace(strings.TrimPrefix(trimmed, "IdentityFile"))
			}
		}
		before = lines[:startIdx]
		if endIdx+1 < len(lines) {
			after = lines[endIdx+1:]
		}
		return before, after, identityFile, nil
	}

	// No managed block — look for legacy "Host github.com" to migrate
	hostIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, "Host github.com") {
			hostIdx = i
			break
		}
	}

	if hostIdx >= 0 {
		// Find end of this Host block (next Host line or EOF)
		blockEnd := len(lines)
		for i := hostIdx + 1; i < len(lines); i++ {
			trimmed := strings.TrimSpace(lines[i])
			if strings.HasPrefix(trimmed, "Host ") || strings.HasPrefix(trimmed, "Match ") {
				blockEnd = i
				break
			}
		}

		// Extract IdentityFile from legacy block
		for _, line := range lines[hostIdx:blockEnd] {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "IdentityFile ") {
				identityFile = strings.TrimSpace(strings.TrimPrefix(trimmed, "IdentityFile"))
			}
		}

		// Back up config before migration
		backupSSHConfig(path)

		before = lines[:hostIdx]
		if blockEnd < len(lines) {
			after = lines[blockEnd:]
		}
		return before, after, identityFile, nil
	}

	// No github.com block at all — everything is "before"
	return lines, nil, "", nil
}

func writeSSHConfig(path string, before []string, sshKeyPath string, after []string) error {
	// Ensure ~/.ssh exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating SSH dir: %w", err)
	}

	// Build managed block
	managed := []string{
		markerStart,
		"Host github.com",
		"  HostName github.com",
		"  User git",
		"  AddKeysToAgent yes",
		"  UseKeychain yes",
		"  IdentityFile " + sshKeyPath,
		"  IdentitiesOnly yes",
		markerEnd,
	}

	// Write to temp file then rename (atomic)
	tmpPath := path + ".git-switch-tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	w := bufio.NewWriter(f)
	// Write before section
	for _, line := range before {
		fmt.Fprintln(w, line)
	}
	// Add blank line before managed block if before section exists and doesn't end with blank
	if len(before) > 0 && strings.TrimSpace(before[len(before)-1]) != "" {
		fmt.Fprintln(w)
	}
	// Write managed block
	for _, line := range managed {
		fmt.Fprintln(w, line)
	}
	// Write after section
	if len(after) > 0 {
		fmt.Fprintln(w)
		for _, line := range after {
			fmt.Fprintln(w, line)
		}
	}

	if err := w.Flush(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}
	f.Close()

	// Atomic rename
	return os.Rename(tmpPath, path)
}

func backupSSHConfig(path string) {
	backupPath := path + ".git-switch-backup"
	if _, err := os.Stat(backupPath); err == nil {
		return // backup already exists
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	os.WriteFile(backupPath, data, 0o644)
}
