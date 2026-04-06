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
	rootCmd.AddCommand(showCmd)
}

var showCmd = &cobra.Command{
	Use:               "show [account]",
	Short:             "Show public SSH and GPG keys for an account",
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

		var name string
		if len(args) == 1 {
			name = args[0]
		} else {
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
			idx := ui.Select("Which account?", options)
			if idx < 0 {
				return fmt.Errorf("no account selected")
			}
			name = names[idx]
		}

		acct, ok := cfg.Accounts[name]
		if !ok {
			return fmt.Errorf("account %q not found", name)
		}

		fmt.Printf("Account: %s\n", name)
		fmt.Printf("  %s <%s>\n\n", acct.Name, acct.Email)

		// SSH public key
		keyPath, err := config.ExpandPath(acct.SSHKey)
		if err != nil {
			return err
		}
		pubKey, err := ssh.ReadPublicKey(keyPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "SSH key: %v\n", err)
		} else {
			fmt.Println("--- SSH Public Key (https://github.com/settings/ssh/new) ---")
			fmt.Println(pubKey)
			fmt.Println()
		}

		// GPG public key
		if acct.GPGKey != "" {
			gpgPub, err := gpg.ExportPublicKey(acct.GPGKey)
			if err != nil {
				fmt.Fprintf(os.Stderr, "GPG key: %v\n", err)
			} else {
				fmt.Println("--- GPG Public Key (https://github.com/settings/gpg/new) ---")
				fmt.Println(gpgPub)
			}
		} else {
			fmt.Println("No GPG key configured for this account.")
		}

		return nil
	},
}
