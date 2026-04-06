package cmd

import (
	"fmt"

	"github.com/ishan-sharma-me/git-switch/internal/config"
	"github.com/ishan-sharma-me/git-switch/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:               "remove <account>",
	Short:             "Stop managing an account (keys are not deleted)",
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
		fmt.Println("\nKeys will NOT be deleted from disk.")

		if !ui.Confirm("Remove this account from git-switch?", false) {
			return nil
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
