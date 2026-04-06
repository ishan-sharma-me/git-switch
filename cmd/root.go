package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ishan-sharma-me/git-switch/internal/config"
	"github.com/ishan-sharma-me/git-switch/internal/git"
	"github.com/ishan-sharma-me/git-switch/internal/gpg"
	"github.com/ishan-sharma-me/git-switch/internal/ssh"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "git-switch [account]",
	Short: "Manage multiple Git SSH/GPG identities",
	Long: `git-switch lets you switch between multiple GitHub identities,
each with its own SSH key, git user config, and optional GPG signing key.

Usage:
  git-switch <account>     Switch to an account
  git-switch list          List all managed accounts
  git-switch add           Import an existing SSH key
  git-switch create        Generate new SSH/GPG keys
  git-switch set-local <n> Pin a repo to an account (local git config)
  git-switch test          Test all accounts
  git-switch remove <name> Remove an account
  git-switch reset <name>  Regenerate keys for an account`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeAccountNames,
	SilenceUsage:      true,
	SilenceErrors:     true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return listCmd.RunE(cmd, args)
		}
		return runSwitch(args[0])
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func runSwitch(accountName string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	acct, ok := cfg.Accounts[accountName]
	if !ok {
		// Suggest similar names
		names := cfg.AccountNames()
		if len(names) > 0 {
			return fmt.Errorf("account %q not found. Available: %s", accountName, strings.Join(names, ", "))
		}
		return fmt.Errorf("account %q not found. Run 'git-switch add' to get started", accountName)
	}

	// Verify SSH key exists
	keyPath, err := config.ExpandPath(acct.SSHKey)
	if err != nil {
		return err
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH key not found: %s", acct.SSHKey)
	}

	// Step 1: Known hosts
	fmt.Print("[1/6] Checking known hosts... ")
	if err := ssh.EnsureGitHubKnownHost(); err != nil {
		fmt.Println("warning:", err)
	} else {
		fmt.Println("ok")
	}

	// Step 2: SSH config
	fmt.Print("[2/6] Updating SSH config... ")
	if err := ssh.UpdateGitHubIdentity(acct.SSHKey); err != nil {
		return fmt.Errorf("updating SSH config: %w", err)
	}
	fmt.Println("ok")

	// Step 3: SSH agent
	fmt.Print("[3/6] Loading SSH key... ")
	if err := ssh.AddKeyToAgent(keyPath); err != nil {
		fmt.Println("warning:", err)
	} else {
		fmt.Println("ok")
	}

	// Step 4: Git config
	fmt.Print("[4/6] Updating git config... ")
	if err := git.SetGlobalUser(acct.Name, acct.Email); err != nil {
		return fmt.Errorf("updating git config: %w", err)
	}
	fmt.Printf("%s <%s>\n", acct.Name, acct.Email)

	// Step 5: GPG
	fmt.Print("[5/6] Configuring GPG... ")
	if acct.GPGKey != "" {
		if err := gpg.ValidateKey(acct.GPGKey); err != nil {
			fmt.Printf("warning: %v\n", err)
		}
		if err := git.SetGlobalGPG(acct.GPGKey); err != nil {
			return fmt.Errorf("updating GPG config: %w", err)
		}
		fmt.Printf("signing with %s\n", acct.GPGKey)
	} else {
		if err := git.DisableGPG(); err != nil {
			return fmt.Errorf("disabling GPG: %w", err)
		}
		fmt.Println("signing disabled")
	}

	// Step 6: Test connection
	fmt.Print("[6/6] Testing connection... ")
	username, err := ssh.TestGitHubAuth(keyPath)
	if err != nil {
		fmt.Printf("warning: %v\n", err)
	} else {
		fmt.Printf("authenticated as %s\n", username)
	}

	// Save active
	cfg.Active = accountName
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("\nSwitched to %s\n", accountName)
	return nil
}

func completeAccountNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var completions []string
	for name, acct := range cfg.Accounts {
		if strings.HasPrefix(name, toComplete) {
			completions = append(completions, fmt.Sprintf("%s\t%s <%s>", name, acct.Name, acct.Email))
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
