package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ishan-sharma-me/git-switch/internal/config"
	gitcfg "github.com/ishan-sharma-me/git-switch/internal/git"
	"github.com/ishan-sharma-me/git-switch/internal/gpg"
	"github.com/ishan-sharma-me/git-switch/internal/ssh"
	"github.com/ishan-sharma-me/git-switch/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Import an existing SSH key as a managed account",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// Discover SSH keys
		keys, err := ssh.DiscoverKeys()
		if err != nil {
			return fmt.Errorf("discovering SSH keys: %w", err)
		}
		if len(keys) == 0 {
			fmt.Println("No SSH keys found in ~/.ssh/")
			fmt.Println("Run 'git-switch create' to generate a new key pair.")
			return nil
		}

		// Build selection list
		options := make([]string, len(keys))
		for i, k := range keys {
			short := config.ShortenPath(k.Path)
			options[i] = fmt.Sprintf("%s  %s  %s", short, k.Fingerprint, k.Comment)
		}

		idx := ui.Select("Select an SSH key:", options)
		if idx < 0 {
			return fmt.Errorf("no key selected")
		}
		selected := keys[idx]

		// Account name
		defaultName := strings.TrimSuffix(filepath.Base(selected.Path), filepath.Ext(selected.Path))
		fmt.Println("\nChoose a short name for this account (e.g. your GitHub username).")
		fmt.Println("This is what you'll type to switch: git-switch <name>")
		accountName := ui.Prompt("Account name", defaultName)
		if accountName == "" {
			return fmt.Errorf("account name is required")
		}
		if _, exists := cfg.Accounts[accountName]; exists {
			return fmt.Errorf("account %q already exists", accountName)
		}

		// Git user info
		currentName, currentEmail, _ := gitcfg.GetGlobalUser()
		userName := ui.Prompt("Git user.name", currentName)
		userEmail := ui.Prompt("Git user.email", currentEmail)

		// GPG key (optional)
		var gpgKeyID string
		gpgKeys, gpgErr := gpg.ListSecretKeys()
		if gpgErr == nil && len(gpgKeys) > 0 {
			if ui.Confirm("Associate a GPG signing key?", false) {
				gpgOptions := make([]string, len(gpgKeys))
				for i, k := range gpgKeys {
					gpgOptions[i] = fmt.Sprintf("%s  %s", k.KeyID, k.UserID)
				}
				gpgIdx := ui.Select("Select a GPG key:", gpgOptions)
				if gpgIdx >= 0 {
					gpgKeyID = gpgKeys[gpgIdx].KeyID
				}
			}
		}

		// Save
		cfg.Accounts[accountName] = &config.Account{
			SSHKey: config.ShortenPath(selected.Path),
			Name:   userName,
			Email:  userEmail,
			GPGKey: gpgKeyID,
		}

		// If first account, make it active
		if len(cfg.Accounts) == 1 {
			cfg.Active = accountName
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("\nAdded account %q\n", accountName)
		if len(cfg.Accounts) == 1 {
			fmt.Printf("Run 'git-switch %s' to activate it.\n", accountName)
		}
		return nil
	},
}
