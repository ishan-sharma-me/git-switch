package cmd

import (
	"fmt"
	"strings"

	"github.com/ishan-sharma-me/git-switch/internal/config"
	"github.com/ishan-sharma-me/git-switch/internal/git"
	"github.com/ishan-sharma-me/git-switch/internal/gpg"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(setLocalCmd)
}

var setLocalCmd = &cobra.Command{
	Use:   "set-local <account>",
	Short: "Set repo-level git identity to a managed account",
	Long: `Sets git user.name, user.email, and GPG signing config at the
repo level (git config --local) so this repository always uses
the specified account regardless of the global git-switch profile.

Must be run from inside a git repository.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeAccountNames,
	SilenceUsage:      true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !git.IsInsideWorkTree() {
			return fmt.Errorf("not inside a git repository")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		accountName := args[0]
		acct, ok := cfg.Accounts[accountName]
		if !ok {
			names := cfg.AccountNames()
			if len(names) > 0 {
				return fmt.Errorf("account %q not found. Available: %s", accountName, strings.Join(names, ", "))
			}
			return fmt.Errorf("account %q not found. Run 'git-switch add' to get started", accountName)
		}

		// Set local git user
		fmt.Print("Setting local git user... ")
		if err := git.SetLocalUser(acct.Name, acct.Email); err != nil {
			return fmt.Errorf("setting local user: %w", err)
		}
		fmt.Printf("%s <%s>\n", acct.Name, acct.Email)

		// Set local GPG config
		fmt.Print("Configuring local GPG... ")
		if acct.GPGKey != "" {
			if err := gpg.ValidateKey(acct.GPGKey); err != nil {
				fmt.Printf("warning: %v\n", err)
			}
			if err := git.SetLocalGPG(acct.GPGKey); err != nil {
				return fmt.Errorf("setting local GPG: %w", err)
			}
			fmt.Printf("signing with %s\n", acct.GPGKey)
		} else {
			if err := git.DisableLocalGPG(); err != nil {
				return fmt.Errorf("disabling local GPG: %w", err)
			}
			fmt.Println("signing disabled")
		}

		fmt.Printf("\nThis repository will now always commit as %s <%s>\n", acct.Name, acct.Email)
		return nil
	},
}
