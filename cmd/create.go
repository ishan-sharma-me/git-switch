package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ishan-sharma-me/git-switch/internal/config"
	"github.com/ishan-sharma-me/git-switch/internal/gpg"
	"github.com/ishan-sharma-me/git-switch/internal/ssh"
	"github.com/ishan-sharma-me/git-switch/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Generate new SSH/GPG keys for a new account",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// Account name
		fmt.Println("Choose a short name for this account (e.g. your GitHub username).")
		fmt.Println("This is what you'll type to switch: git-switch <name>")
		accountName := ui.Prompt("Account name", "")
		if accountName == "" {
			return fmt.Errorf("account name is required")
		}
		if _, exists := cfg.Accounts[accountName]; exists {
			return fmt.Errorf("account %q already exists", accountName)
		}

		// Git user info
		userName := ui.Prompt("Git user.name", "")
		if userName == "" {
			return fmt.Errorf("user.name is required")
		}
		userEmail := ui.Prompt("Git user.email", "")
		if userEmail == "" {
			return fmt.Errorf("user.email is required")
		}

		// Generate SSH key
		home, _ := os.UserHomeDir()
		defaultKeyPath := filepath.Join(home, ".ssh", accountName)
		keyPath := ui.Prompt("SSH key path", defaultKeyPath)

		// Check if key already exists
		if _, err := os.Stat(keyPath); err == nil {
			if !ui.Confirm(fmt.Sprintf("Key %s already exists. Overwrite?", keyPath), false) {
				return fmt.Errorf("aborted")
			}
		}

		fmt.Println("Generating SSH key...")
		if err := ssh.GenerateKey(keyPath, userEmail); err != nil {
			return fmt.Errorf("generating SSH key: %w", err)
		}
		fmt.Printf("Created: %s\n", keyPath)

		// Add to agent
		if err := ssh.AddKeyToAgent(keyPath); err != nil {
			fmt.Printf("Warning: could not add to agent: %v\n", err)
		}

		// Show public key
		pubKey, err := ssh.ReadPublicKey(keyPath)
		if err != nil {
			return err
		}
		fmt.Println("\n--- Public SSH Key (add to GitHub) ---")
		fmt.Println(pubKey)
		fmt.Println("--------------------------------------")
		fmt.Println("Add this key at: https://github.com/settings/ssh/new")
		ui.WaitForEnter("\nPress Enter when done...")

		// GPG key (optional)
		var gpgKeyID string
		if ui.Confirm("Generate a GPG signing key?", false) {
			fmt.Println("Generating GPG key (RSA 4096)...")
			keyID, err := gpg.GenerateKey(userName, userEmail)
			if err != nil {
				fmt.Printf("Warning: GPG key generation failed: %v\n", err)
			} else {
				gpgKeyID = keyID
				fmt.Printf("GPG key created: %s\n", keyID)

				// Show GPG public key
				gpgPub, err := gpg.ExportPublicKey(keyID)
				if err == nil {
					fmt.Println("\n--- Public GPG Key (add to GitHub) ---")
					fmt.Println(gpgPub)
					fmt.Println("--------------------------------------")
					fmt.Println("Add this key at: https://github.com/settings/gpg/new")
					ui.WaitForEnter("\nPress Enter when done...")
				}
			}
		}

		// Save account
		cfg.Accounts[accountName] = &config.Account{
			SSHKey: config.ShortenPath(keyPath),
			Name:   userName,
			Email:  userEmail,
			GPGKey: gpgKeyID,
		}
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("\nCreated account %q\n", accountName)

		// Offer to switch
		if ui.Confirm("Switch to this account now?", true) {
			return runSwitch(accountName)
		}

		return nil
	},
}
