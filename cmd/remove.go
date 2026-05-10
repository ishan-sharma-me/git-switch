package cmd

import (
	"fmt"

	"github.com/aanya-send-help/git-switch/internal/config"
	"github.com/aanya-send-help/git-switch/internal/gh"
	"github.com/aanya-send-help/git-switch/internal/ssh"
	"github.com/aanya-send-help/git-switch/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:               "remove <account>",
	Short:             "Stop managing an account (also deletes its keys from GitHub)",
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
		if acct.GitHubLogin != "" {
			fmt.Printf("  GitHub: @%s\n", acct.GitHubLogin)
		}
		fmt.Println("\nLocal key files on disk are NOT deleted.")

		if !ui.Confirm("Remove this account from git-switch?", false) {
			return nil
		}

		if acct.GitHubLogin != "" {
			if ui.Confirm("Also delete the SSH and GPG keys from GitHub?", true) {
				deleteRemoteKeys(acct)
			}
		}

		delete(cfg.Accounts, name)
		if cfg.Active == name {
			cfg.Active = ""
			fmt.Println("Note: this was the active account. Run 'git-switch <name>' to switch.")
		}

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Printf("Removed %q\n", name)
		return nil
	},
}

// deleteRemoteKeys is best-effort: any failure just prints a warning so the
// local removal still succeeds. The user can clean up stale GitHub keys
// manually if needed.
func deleteRemoteKeys(acct *config.Account) {
	if err := gh.SwitchAccount(acct.GitHubLogin); err != nil {
		fmt.Printf("Warning: could not switch gh to @%s: %v\n", acct.GitHubLogin, err)
		return
	}

	keyPath, err := config.ExpandPath(acct.SSHKey)
	if err == nil {
		if pub, perr := ssh.ReadPublicKey(keyPath); perr == nil {
			if remote, lerr := gh.ListSSHKeys(); lerr == nil {
				if match := gh.MatchSSHKeyByContent(remote, pub); match != nil {
					if derr := gh.DeleteSSHKey(match.ID); derr != nil {
						fmt.Printf("Warning: deleting GitHub SSH key %d: %v\n", match.ID, derr)
					} else {
						fmt.Printf("Deleted SSH key from GitHub (id %d).\n", match.ID)
					}
				}
			}
		}
	}

	if acct.GPGKey != "" {
		if remote, lerr := gh.ListGPGKeys(); lerr == nil {
			if match := gh.MatchGPGKeyByID(remote, acct.GPGKey); match != nil {
				if derr := gh.DeleteGPGKey(match.ID); derr != nil {
					fmt.Printf("Warning: deleting GitHub GPG key %d: %v\n", match.ID, derr)
				} else {
					fmt.Printf("Deleted GPG key from GitHub (id %d).\n", match.ID)
				}
			}
		}
	}
}
