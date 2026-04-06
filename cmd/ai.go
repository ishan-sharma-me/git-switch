package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	aiMarkerStart = "<!-- git-switch-start -->"
	aiMarkerEnd   = "<!-- git-switch-end -->"

	aiContent = `<!-- git-switch-start -->
## Git Identity Management

This system has ` + "`git-switch`" + ` installed for managing multiple GitHub SSH/GPG identities.

Available commands:
- ` + "`git-switch <account-name>`" + ` — Switch active GitHub identity (SSH key, git user, GPG)
- ` + "`git-switch list`" + ` — Show all managed accounts and which is active
- ` + "`git-switch test`" + ` — Test SSH connectivity and GPG signing for all accounts
- ` + "`git-switch add`" + ` — Import an existing SSH key as a managed account (interactive)
- ` + "`git-switch create`" + ` — Generate new SSH/GPG keys for a new account (interactive)
- ` + "`git-switch remove <name>`" + ` — Remove an account from management (keys not deleted)
- ` + "`git-switch reset <name>`" + ` — Regenerate SSH/GPG keys for an account

When the user wants to work on a repo under a different GitHub account, use ` + "`git-switch <name>`" + ` to switch identity before cloning or pushing.
Use ` + "`git-switch list`" + ` to see available accounts. Use ` + "`git-switch test`" + ` to verify all accounts are working.
<!-- git-switch-end -->`
)

func init() {
	aiCmd.Flags().Bool("remove", false, "Remove Claude Code integration")
	rootCmd.AddCommand(aiCmd)
}

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Install Claude Code integration (global CLAUDE.md)",
	Long: `Installs git-switch instructions into ~/.claude/CLAUDE.md so that
Claude Code knows how to manage your Git identities.

Use --remove to uninstall.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		remove, _ := cmd.Flags().GetBool("remove")

		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		claudeDir := filepath.Join(home, ".claude")
		claudeMD := filepath.Join(claudeDir, "CLAUDE.md")

		if remove {
			return removeAIIntegration(claudeMD)
		}
		return installAIIntegration(claudeDir, claudeMD)
	},
}

func installAIIntegration(claudeDir, claudeMD string) error {
	// Ensure ~/.claude exists
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return fmt.Errorf("creating ~/.claude: %w", err)
	}

	// Read existing content
	existing, _ := os.ReadFile(claudeMD)
	content := string(existing)

	// Check if already installed — replace if so
	if strings.Contains(content, aiMarkerStart) {
		content = removeMarkedSection(content)
		fmt.Println("Updating existing Claude Code integration...")
	} else {
		fmt.Println("Installing Claude Code integration...")
	}

	// Append our section
	if content != "" && !strings.HasSuffix(content, "\n\n") {
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n"
	}
	content += aiContent + "\n"

	if err := os.WriteFile(claudeMD, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing CLAUDE.md: %w", err)
	}

	fmt.Printf("Installed to %s\n", claudeMD)
	fmt.Println("Claude Code will now know about git-switch in all projects.")
	return nil
}

func removeAIIntegration(claudeMD string) error {
	data, err := os.ReadFile(claudeMD)
	if os.IsNotExist(err) {
		fmt.Println("Nothing to remove — CLAUDE.md does not exist.")
		return nil
	}
	if err != nil {
		return err
	}

	content := string(data)
	if !strings.Contains(content, aiMarkerStart) {
		fmt.Println("Nothing to remove — git-switch section not found in CLAUDE.md.")
		return nil
	}

	content = removeMarkedSection(content)

	// Clean up extra blank lines
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}
	content = strings.TrimSpace(content)
	if content != "" {
		content += "\n"
	}

	if err := os.WriteFile(claudeMD, []byte(content), 0o644); err != nil {
		return err
	}

	fmt.Println("Removed git-switch section from CLAUDE.md.")
	return nil
}

func removeMarkedSection(content string) string {
	startIdx := strings.Index(content, aiMarkerStart)
	endIdx := strings.Index(content, aiMarkerEnd)
	if startIdx < 0 || endIdx < 0 {
		return content
	}
	return content[:startIdx] + content[endIdx+len(aiMarkerEnd):]
}
