package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/ishan-sharma-me/git-switch/internal/config"
	"github.com/ishan-sharma-me/git-switch/internal/gpg"
	"github.com/ishan-sharma-me/git-switch/internal/ssh"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(testCmd)
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test SSH and GPG for all managed accounts",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if len(cfg.Accounts) == 0 {
			fmt.Println("No accounts configured. Run 'git-switch add' to get started.")
			return nil
		}

		fmt.Println("Testing all accounts...")
		fmt.Println()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "  ACCOUNT\tSSH KEY\tSSH AUTH\tGPG\n")
		fmt.Fprintf(w, "  ───────\t───────\t────────\t───\n")

		allOK := true

		for _, name := range cfg.AccountNames() {
			acct := cfg.Accounts[name]
			marker := " "
			if name == cfg.Active {
				marker = "*"
			}

			// Expand key path
			keyPath, err := config.ExpandPath(acct.SSHKey)
			if err != nil {
				fmt.Fprintf(w, "%s %s\t%s\terror\terror\n", marker, name, acct.SSHKey)
				allOK = false
				continue
			}

			// Check key file exists
			if _, err := os.Stat(keyPath); os.IsNotExist(err) {
				fmt.Fprintf(w, "%s %s\t%s\tkey missing\t-\n", marker, name, acct.SSHKey)
				allOK = false
				continue
			}

			// Test SSH auth
			sshResult := "ok"
			username, err := ssh.TestGitHubAuth(keyPath)
			if err != nil {
				sshResult = "FAIL"
				allOK = false
			} else {
				sshResult = username
			}

			// Test GPG
			gpgResult := "no key"
			if acct.GPGKey != "" {
				if err := gpg.ValidateKey(acct.GPGKey); err != nil {
					gpgResult = "FAIL"
					allOK = false
				} else {
					gpgResult = "ok " + acct.GPGKey
				}
			}

			fmt.Fprintf(w, "%s %s\t%s\t%s\t%s\n", marker, name, acct.SSHKey, sshResult, gpgResult)
		}

		w.Flush()

		if allOK {
			fmt.Println("\nAll accounts OK")
		} else {
			fmt.Println("\nSome accounts have issues")
		}
		return nil
	},
}
