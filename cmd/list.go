package cmd

import (
	"fmt"

	"github.com/ishan-sharma-me/git-switch/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all managed accounts",
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

		for _, name := range cfg.AccountNames() {
			acct := cfg.Accounts[name]
			marker := "  "
			if name == cfg.Active {
				marker = "* "
			}
			fmt.Printf("%s%s\n", marker, name)
			fmt.Printf("    %s <%s>\n", acct.Name, acct.Email)
			fmt.Printf("    SSH: %s\n", acct.SSHKey)
			if acct.GPGKey != "" {
				fmt.Printf("    GPG: %s\n", acct.GPGKey)
			}
		}
		return nil
	},
}
