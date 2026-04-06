# git-switch

> **Disclaimer:** This project was coded with AI assistance. Please be cautious when using it, especially if you are an open source maintainer with critical repo access or commit signing responsibilities. A comprehensive security audit is pending. Review the source and test in a safe environment before relying on it for production workflows.

A CLI tool for managing multiple Git SSH/GPG identities. Switch between GitHub accounts with a single command.

## Why?

If you work across multiple GitHub accounts (personal, work, consulting), you know the pain: manually editing `~/.ssh/config`, updating `git config`, swapping GPG keys, and praying you didn't push to the wrong repo. `git-switch` automates all of it.

One command switches your entire identity:
- SSH config (which key GitHub sees)
- SSH agent (key loaded and ready)
- Git user config (name, email)
- GPG signing (enabled/disabled per account)
- Known hosts (github.com always trusted)
- Connectivity test (confirms it worked)

## Install

**Quick install** (macOS & Linux):

```bash
curl -fsSL https://ishan-sharma-me.github.io/git-switch/install.sh | bash
```

This downloads the latest binary for your platform, installs it to `~/.local/bin/`, sets up shell completions, and configures your PATH. Open a new terminal tab after installing.

**From source** (requires Go):

```bash
git clone https://github.com/ishan-sharma-me/git-switch.git
cd git-switch
make install
```

## Usage

```bash
# Add an existing SSH key as a managed account
git-switch add

# Create a new account with fresh SSH/GPG keys
git-switch create

# Switch to an account
git-switch personal

# List all accounts (also the default when you run git-switch with no args)
git-switch list

# Test SSH and GPG for all accounts
git-switch test

# Edit an account's name or details
git-switch edit personal

# Regenerate keys for an account
git-switch reset personal

# Remove an account (keys stay on disk)
git-switch remove personal
```

## What happens when you switch

```
$ git-switch work

[1/6] Checking known hosts... ok
[2/6] Updating SSH config... ok
[3/6] Loading SSH key... ok
[4/6] Updating git config... Jane Doe <jane@company.com>
[5/6] Configuring GPG... signing with ABCD1234EF567890
[6/6] Testing connection... authenticated as janedoe-work

Switched to work
```

## How it works

`git-switch` manages a single block in your `~/.ssh/config` using comment markers:

```
# git-switch-managed-start
Host github.com
  HostName github.com
  User git
  AddKeysToAgent yes
  UseKeychain yes
  IdentityFile ~/.ssh/id_work
  IdentitiesOnly yes
# git-switch-managed-end
```

Everything outside these markers is never touched. Your other SSH hosts (servers, infrastructure, etc.) are safe.

Account data is stored in `~/.config/git-switch/config.yaml`.

## Shell completions

Completions are installed automatically by both install methods. Tab-complete account names in zsh, bash, and fish.

To set up manually:

```bash
# Zsh
mkdir -p ~/.local/share/zsh/site-functions
git-switch completion zsh > ~/.local/share/zsh/site-functions/_git-switch

# Bash
git-switch completion bash > ~/.local/share/bash-completion/completions/git-switch

# Fish
git-switch completion fish > ~/.config/fish/completions/git-switch.fish
```

## Claude Code integration

```bash
git-switch ai          # Install global instructions
git-switch ai --remove # Uninstall
```

Adds git-switch awareness to `~/.claude/CLAUDE.md` so Claude Code knows how to manage your identities when working across repos.

## Uninstall

**Quick uninstall** (reverses the curl install):

```bash
curl -fsSL https://ishan-sharma-me.github.io/git-switch/uninstall.sh | bash
```

This removes the binary, shell completions, and Claude Code integration. Your config (`~/.config/git-switch/`) and SSH keys are not deleted.

**From source:**

```bash
make uninstall            # Remove binary
make uninstall-completion # Remove shell completions
make uninstall-path       # Remove PATH entry from shell config
```

## License

MIT
