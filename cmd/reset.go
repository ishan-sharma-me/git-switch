package cmd

import (
	"fmt"
	"os"

	"github.com/ishan-sharma-me/git-switch/internal/config"
	"github.com/ishan-sharma-me/git-switch/internal/gpg"
	"github.com/ishan-sharma-me/git-switch/internal/ssh"
	"github.com/ishan-sharma-me/git-switch/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(resetCmd)
}

var resetCmd = &cobra.Command{
	Use:               "reset <account>",
	Short:             "Regenerate SSH/GPG keys for an account",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeAccountNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		name := args[0]
		acct, ok := cfg.Accounts[name]
		if !ok {
			return fmt.Errorf("account %q not found", name)
		}

		fmt.Printf("Account: %s\n", name)
		fmt.Printf("  %s <%s>\n", acct.Name, acct.Email)
		fmt.Printf("  SSH: %s\n", acct.SSHKey)
		if acct.GPGKey != "" {
			fmt.Printf("  GPG: %s\n", acct.GPGKey)
		}

		// What to reset
		resetSSH := ui.Confirm("\nRegenerate SSH key?", true)
		resetGPG := ui.Confirm("Regenerate GPG key?", acct.GPGKey != "")

		if !resetSSH && !resetGPG {
			fmt.Println("Nothing to do.")
			return nil
		}

		if !ui.Confirm("\nThis will overwrite existing keys. Continue?", false) {
			return nil
		}

		// Reset SSH key
		if resetSSH {
			keyPath, err := config.ExpandPath(acct.SSHKey)
			if err != nil {
				return err
			}

			// Remove old key files
			os.Remove(keyPath)
			os.Remove(keyPath + ".pub")

			fmt.Println("\nGenerating new SSH key...")
			if err := ssh.GenerateKey(keyPath, acct.Email); err != nil {
				return fmt.Errorf("generating SSH key: %w", err)
			}

			// Add to agent
			if err := ssh.AddKeyToAgent(keyPath); err != nil {
				fmt.Printf("Warning: could not add to agent: %v\n", err)
			}

			// Show public key
			pubKey, err := ssh.ReadPublicKey(keyPath)
			if err != nil {
				return err
			}
			fmt.Println("\n--- New Public SSH Key ---")
			fmt.Println(pubKey)
			fmt.Println("-------------------------")
			fmt.Println("Update this key at: https://github.com/settings/keys")
			fmt.Println("  1. Delete the old key")
			fmt.Println("  2. Add the new key above")
			ui.WaitForEnter("\nPress Enter when done...")
		}

		// Reset GPG key
		if resetGPG {
			fmt.Println("\nGenerating new GPG key (RSA 4096)...")
			keyID, err := gpg.GenerateKey(acct.Name, acct.Email)
			if err != nil {
				return fmt.Errorf("generating GPG key: %w", err)
			}
			acct.GPGKey = keyID
			fmt.Printf("GPG key created: %s\n", keyID)

			gpgPub, err := gpg.ExportPublicKey(keyID)
			if err == nil {
				fmt.Println("\n--- New Public GPG Key ---")
				fmt.Println(gpgPub)
				fmt.Println("--------------------------")
				fmt.Println("Update this key at: https://github.com/settings/gpg/new")
				ui.WaitForEnter("\nPress Enter when done...")
			}
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		// Test connectivity
		fmt.Println("\nTesting connection...")
		keyPath, _ := config.ExpandPath(acct.SSHKey)
		username, err := ssh.TestGitHubAuth(keyPath)
		if err != nil {
			fmt.Printf("Warning: SSH test failed: %v\n", err)
			fmt.Println("Make sure you've added the new public key to GitHub.")
		} else {
			fmt.Printf("Authenticated as: %s\n", username)
		}

		if acct.GPGKey != "" {
			if err := gpg.ValidateKey(acct.GPGKey); err != nil {
				fmt.Printf("Warning: GPG test failed: %v\n", err)
			} else {
				fmt.Println("GPG signing: ok")
			}
		}

		fmt.Printf("\nReset complete for %s\n", name)
		return nil
	},
}
