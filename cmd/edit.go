package cmd

import (
	"fmt"

	"github.com/ishan-sharma-me/git-switch/internal/config"
	"github.com/ishan-sharma-me/git-switch/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(editCmd)
}

var editCmd = &cobra.Command{
	Use:               "edit [account]",
	Short:             "Edit an account's name or details",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeAccountNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if len(cfg.Accounts) == 0 {
			fmt.Println("No accounts configured. Run 'git-switch add' to get started.")
			return nil
		}

		var oldName string
		if len(args) == 1 {
			oldName = args[0]
		} else {
			// Interactive selection
			names := cfg.AccountNames()
			options := make([]string, len(names))
			for i, n := range names {
				a := cfg.Accounts[n]
				marker := " "
				if n == cfg.Active {
					marker = "*"
				}
				options[i] = fmt.Sprintf("%s %s  %s <%s>", marker, n, a.Name, a.Email)
			}
			idx := ui.Select("Which account do you want to edit?", options)
			if idx < 0 {
				return fmt.Errorf("no account selected")
			}
			oldName = names[idx]
		}

		acct, ok := cfg.Accounts[oldName]
		if !ok {
			return fmt.Errorf("account %q not found", oldName)
		}

		fmt.Printf("Editing account: %s\n", oldName)
		fmt.Printf("  %s <%s>\n", acct.Name, acct.Email)
		fmt.Printf("  SSH: %s\n", acct.SSHKey)
		if acct.GPGKey != "" {
			fmt.Printf("  GPG: %s\n", acct.GPGKey)
		}
		fmt.Println()

		newName := ui.Prompt("Account name (used for git-switch <name>)", oldName)
		acct.Name = ui.Prompt("Git user.name", acct.Name)
		acct.Email = ui.Prompt("Git user.email", acct.Email)
		acct.GPGKey = ui.Prompt("GPG key ID (leave empty to disable)", acct.GPGKey)

		// Rename if account name changed
		if newName != oldName {
			if _, exists := cfg.Accounts[newName]; exists {
				return fmt.Errorf("account %q already exists", newName)
			}
			delete(cfg.Accounts, oldName)
			cfg.Accounts[newName] = acct
			if cfg.Active == oldName {
				cfg.Active = newName
			}
			fmt.Printf("\nRenamed %q -> %q\n", oldName, newName)
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Println("Account updated.")
		return nil
	},
}
