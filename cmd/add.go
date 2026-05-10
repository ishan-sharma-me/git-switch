package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aanya-send-help/git-switch/internal/config"
	"github.com/aanya-send-help/git-switch/internal/gh"
	"github.com/aanya-send-help/git-switch/internal/gpg"
	"github.com/aanya-send-help/git-switch/internal/ssh"
	"github.com/aanya-send-help/git-switch/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a GitHub account (logs you in via gh, generates and uploads keys)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		fmt.Println("Choose a short name for this account (e.g. work, personal).")
		fmt.Println("This is what you'll type to switch: git-switch <name>")
		alias := ui.Prompt("Account alias", "")
		if alias == "" {
			return fmt.Errorf("account alias is required")
		}
		if _, exists := cfg.Accounts[alias]; exists {
			return fmt.Errorf("account %q already exists", alias)
		}

		fmt.Println("\nLogging in to GitHub for this account.")
		fmt.Println("A browser window will open. Sign in with the GitHub user this alias should map to.")
		ui.WaitForEnter("Press Enter to continue (Ctrl-C to cancel)...")

		bind, err := gh.BindAccount()
		if err != nil {
			return err
		}
		fmt.Printf("Logged in as @%s.\n", bind.Login)

		userName := ui.Prompt("Git user.name", bind.Name)
		userEmail := ui.Prompt("Git user.email", bind.Email)
		if userName == "" || userEmail == "" {
			return fmt.Errorf("name and email are required")
		}

		keyPath, err := chooseSSHKey(alias, userEmail)
		if err != nil {
			return err
		}

		if err := gh.AddSSHKey(keyPath+".pub", sshKeyTitle(alias)); err != nil {
			return err
		}
		fmt.Println("Uploaded SSH key to GitHub.")

		var gpgKeyID string
		if ui.Confirm("\nGenerate a GPG signing key for verified commits?", true) {
			fmt.Println("Generating GPG key (RSA 4096)... this can take 30-60 seconds.")
			id, gerr := gpg.GenerateKey(userName, userEmail)
			if gerr != nil {
				fmt.Printf("Warning: GPG generation failed: %v\n", gerr)
			} else {
				gpgKeyID = id
				armored, perr := gpg.ExportPublicKey(id)
				if perr != nil {
					fmt.Printf("Warning: exporting GPG key: %v\n", perr)
				} else if uperr := gh.AddGPGKey(armored); uperr != nil {
					fmt.Printf("Warning: uploading GPG key: %v\n", uperr)
				} else {
					fmt.Printf("Uploaded GPG key %s to GitHub.\n", id)
				}
			}
		}

		cfg.Accounts[alias] = &config.Account{
			SSHKey:      config.ShortenPath(keyPath),
			Name:        userName,
			Email:       userEmail,
			GPGKey:      gpgKeyID,
			GitHubLogin: bind.Login,
		}
		if len(cfg.Accounts) == 1 {
			cfg.Active = alias
		}
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Printf("\nAdded account %q (linked to @%s).\n", alias, bind.Login)

		if ui.Confirm("Switch to this account now?", true) {
			return runSwitch(alias)
		}
		return nil
	},
}

// chooseSSHKey returns the absolute path to the private SSH key the user wants
// to associate with this alias. Either generates a fresh ed25519 key at
// ~/.ssh/<alias> or lets the user pick an existing one. The returned key has
// already been added to ssh-agent (best-effort).
func chooseSSHKey(alias, email string) (string, error) {
	options := []string{
		"Generate a fresh ed25519 key (recommended)",
		"Use an existing key on this machine",
	}
	idx := ui.Select("\nWhich SSH key should this account use?", options)
	if idx < 0 {
		return "", fmt.Errorf("cancelled")
	}

	if idx == 0 {
		home, _ := os.UserHomeDir()
		defaultKeyPath := filepath.Join(home, ".ssh", alias)
		keyPath := ui.Prompt("SSH key path", defaultKeyPath)
		if _, err := os.Stat(keyPath); err == nil {
			if !ui.Confirm(fmt.Sprintf("Key %s already exists. Overwrite?", keyPath), false) {
				return "", fmt.Errorf("aborted")
			}
			os.Remove(keyPath)
			os.Remove(keyPath + ".pub")
		}
		fmt.Println("Generating SSH key...")
		if err := ssh.GenerateKey(keyPath, email); err != nil {
			return "", fmt.Errorf("generating SSH key: %w", err)
		}
		if err := ssh.AddKeyToAgent(keyPath); err != nil {
			fmt.Printf("Warning: could not add to agent: %v\n", err)
		}
		return keyPath, nil
	}

	// Existing key
	keys, err := ssh.DiscoverKeys()
	if err != nil {
		return "", fmt.Errorf("discovering SSH keys: %w", err)
	}
	if len(keys) == 0 {
		return "", fmt.Errorf("no existing SSH keys found in ~/.ssh/")
	}
	opts := make([]string, len(keys))
	for i, k := range keys {
		opts[i] = fmt.Sprintf("%s  %s  %s", config.ShortenPath(k.Path), k.Fingerprint, k.Comment)
	}
	pickIdx := ui.Select("Select an SSH key:", opts)
	if pickIdx < 0 {
		return "", fmt.Errorf("no key selected")
	}
	if err := ssh.AddKeyToAgent(keys[pickIdx].Path); err != nil {
		fmt.Printf("Warning: could not add to agent: %v\n", err)
	}
	return keys[pickIdx].Path, nil
}

// sshKeyTitle is the title shown on GitHub's SSH-keys page. Combines the alias
// with the local hostname so the user can tell their devices apart.
func sshKeyTitle(alias string) string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "git-switch:" + alias
	}
	host = strings.SplitN(host, ".", 2)[0]
	return fmt.Sprintf("git-switch:%s@%s", alias, host)
}
