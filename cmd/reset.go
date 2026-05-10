package cmd

import (
	"fmt"
	"os"

	"github.com/aanya-send-help/git-switch/internal/config"
	"github.com/aanya-send-help/git-switch/internal/gh"
	"github.com/aanya-send-help/git-switch/internal/gpg"
	"github.com/aanya-send-help/git-switch/internal/ssh"
	"github.com/aanya-send-help/git-switch/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(resetCmd)
}

var resetCmd = &cobra.Command{
	Use:               "reset <account>",
	Short:             "Regenerate SSH/GPG keys for an account (also re-syncs them on GitHub)",
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

		resetSSH := ui.Confirm("\nRegenerate SSH key?", true)
		resetGPG := ui.Confirm("Regenerate GPG key?", acct.GPGKey != "")

		if !resetSSH && !resetGPG {
			fmt.Println("Nothing to do.")
			return nil
		}

		if !ui.Confirm("\nThis will overwrite existing keys. Continue?", false) {
			return nil
		}

		// If we're going to talk to GitHub, switch the gh CLI to the right user up front.
		if acct.GitHubLogin != "" {
			if err := gh.SwitchAccount(acct.GitHubLogin); err != nil {
				fmt.Printf("Warning: gh switch to @%s failed: %v\n", acct.GitHubLogin, err)
				fmt.Println("GitHub-side updates will be skipped; update keys manually if needed.")
				acct.GitHubLogin = "" // prevent gh.* calls below for this run
			}
		}

		// Capture old key material before regen, so we can match-and-delete on GitHub afterwards.
		var oldSSHPub string
		oldGPGKey := acct.GPGKey
		if acct.GitHubLogin != "" && resetSSH {
			if keyPath, err := config.ExpandPath(acct.SSHKey); err == nil {
				oldSSHPub, _ = ssh.ReadPublicKey(keyPath)
			}
		}

		if resetSSH {
			keyPath, err := config.ExpandPath(acct.SSHKey)
			if err != nil {
				return err
			}
			os.Remove(keyPath)
			os.Remove(keyPath + ".pub")

			fmt.Println("\nGenerating new SSH key...")
			if err := ssh.GenerateKey(keyPath, acct.Email); err != nil {
				return fmt.Errorf("generating SSH key: %w", err)
			}
			if err := ssh.AddKeyToAgent(keyPath); err != nil {
				fmt.Printf("Warning: could not add to agent: %v\n", err)
			}

			if acct.GitHubLogin != "" {
				if err := gh.AddSSHKey(keyPath+".pub", "git-switch:"+name); err != nil {
					fmt.Printf("Warning: uploading new SSH key: %v\n", err)
				} else {
					fmt.Println("Uploaded new SSH key to GitHub.")
					if oldSSHPub != "" {
						if remote, lerr := gh.ListSSHKeys(); lerr == nil {
							if match := gh.MatchSSHKeyByContent(remote, oldSSHPub); match != nil {
								if derr := gh.DeleteSSHKey(match.ID); derr != nil {
									fmt.Printf("Warning: deleting old SSH key from GitHub: %v\n", derr)
								} else {
									fmt.Printf("Deleted old SSH key from GitHub (id %d).\n", match.ID)
								}
							}
						}
					}
				}
			} else {
				pubKey, _ := ssh.ReadPublicKey(keyPath)
				fmt.Println("\n--- New Public SSH Key (paste into GitHub) ---")
				fmt.Println(pubKey)
				fmt.Println("----------------------------------------------")
				ui.WaitForEnter("Press Enter when done...")
			}
		}

		if resetGPG {
			fmt.Println("\nGenerating new GPG key (RSA 4096)...")
			keyID, err := gpg.GenerateKey(acct.Name, acct.Email)
			if err != nil {
				return fmt.Errorf("generating GPG key: %w", err)
			}
			acct.GPGKey = keyID
			fmt.Printf("GPG key created: %s\n", keyID)

			armored, _ := gpg.ExportPublicKey(keyID)
			if acct.GitHubLogin != "" {
				if err := gh.AddGPGKey(armored); err != nil {
					fmt.Printf("Warning: uploading new GPG key: %v\n", err)
				} else {
					fmt.Println("Uploaded new GPG key to GitHub.")
					if oldGPGKey != "" && oldGPGKey != keyID {
						if remote, lerr := gh.ListGPGKeys(); lerr == nil {
							if match := gh.MatchGPGKeyByID(remote, oldGPGKey); match != nil {
								if derr := gh.DeleteGPGKey(match.ID); derr != nil {
									fmt.Printf("Warning: deleting old GPG key from GitHub: %v\n", derr)
								} else {
									fmt.Printf("Deleted old GPG key from GitHub (id %d).\n", match.ID)
								}
							}
						}
					}
				}
			} else {
				fmt.Println("\n--- New Public GPG Key (paste into GitHub) ---")
				fmt.Println(armored)
				fmt.Println("----------------------------------------------")
				ui.WaitForEnter("Press Enter when done...")
			}
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Println("\nTesting connection...")
		keyPath, _ := config.ExpandPath(acct.SSHKey)
		username, err := ssh.TestGitHubAuth(keyPath)
		if err != nil {
			fmt.Printf("Warning: SSH test failed: %v\n", err)
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
