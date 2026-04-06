PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
BINARY := git-switch
ZSH_COMP_DIR := $(HOME)/.local/share/zsh/site-functions
BASH_COMP_DIR := $(HOME)/.local/share/bash-completion/completions
FISH_COMP_DIR := $(HOME)/.config/fish/completions
ZSH_FPATH_LINE := fpath=(~/.local/share/zsh/site-functions $$fpath)

.PHONY: build install install-path install-completion uninstall uninstall-path uninstall-completion clean test

build:
	go build -o $(BINARY) .
	go test ./...
	@echo ""
	@echo "Build and tests passed."

install: build
	@mkdir -p $(BINDIR)
	@cp $(BINARY) $(BINDIR)/$(BINARY)
	@chmod 755 $(BINDIR)/$(BINARY)
	@rm -f $(BINARY)
	@echo "Installed $(BINARY) to $(BINDIR)/"
	@echo ""
	@$(MAKE) -s install-path
	@echo ""
	@$(MAKE) -s install-completion

install-path:
	@if echo "$$PATH" | tr ':' '\n' | grep -qx "$(BINDIR)"; then \
		echo "$(BINDIR) is already in your PATH."; \
	else \
		echo "$(BINDIR) is NOT in your PATH."; \
		echo ""; \
		if [ -f "$$HOME/.zshrc" ]; then \
			echo "Run this to fix it:"; \
			echo '  echo '\''export PATH="$$HOME/.local/bin:$$PATH"'\'' >> ~/.zshrc && source ~/.zshrc'; \
		elif [ -f "$$HOME/.bashrc" ]; then \
			echo "Run this to fix it:"; \
			echo '  echo '\''export PATH="$$HOME/.local/bin:$$PATH"'\'' >> ~/.bashrc && source ~/.bashrc'; \
		elif [ -f "$$HOME/.config/fish/config.fish" ]; then \
			echo "Run this to fix it:"; \
			echo "  fish_add_path $(BINDIR)"; \
		else \
			echo "Add this to your shell profile:"; \
			echo '  export PATH="$$HOME/.local/bin:$$PATH"'; \
		fi; \
	fi

install-completion:
	@if [ -f "$$HOME/.zshrc" ]; then \
		mkdir -p $(ZSH_COMP_DIR); \
		$(BINDIR)/$(BINARY) completion zsh > $(ZSH_COMP_DIR)/_git-switch; \
		if grep -Fq '$(ZSH_FPATH_LINE)' "$$HOME/.zshrc"; then \
			echo "Zsh completions updated."; \
		else \
			ZSHRC=$$(cat "$$HOME/.zshrc"); \
			printf '%s\n%s\n' '$(ZSH_FPATH_LINE)' "$$ZSHRC" > "$$HOME/.zshrc"; \
			echo "Zsh completions installed."; \
			echo "Run: source ~/.zshrc  (or open a new terminal)"; \
		fi; \
	elif [ -f "$$HOME/.bashrc" ]; then \
		mkdir -p $(BASH_COMP_DIR); \
		$(BINDIR)/$(BINARY) completion bash > $(BASH_COMP_DIR)/git-switch; \
		echo "Bash completions installed."; \
	elif [ -f "$$HOME/.config/fish/config.fish" ]; then \
		mkdir -p $(FISH_COMP_DIR); \
		$(BINDIR)/$(BINARY) completion fish > $(FISH_COMP_DIR)/git-switch.fish; \
		echo "Fish completions installed."; \
	else \
		echo "Could not detect shell. Manually run:"; \
		echo "  git-switch completion --help"; \
	fi

uninstall:
	@rm -f $(BINDIR)/$(BINARY)
	@echo "Removed $(BINARY) from $(BINDIR)/"

uninstall-path:
	@PATTERN='export PATH="$$HOME/.local/bin:$$PATH"'; \
	if [ -f "$$HOME/.zshrc" ]; then \
		if grep -Fq "$$PATTERN" "$$HOME/.zshrc"; then \
			grep -Fv "$$PATTERN" "$$HOME/.zshrc" > "$$HOME/.zshrc.tmp" && mv "$$HOME/.zshrc.tmp" "$$HOME/.zshrc"; \
			echo "Removed git-switch PATH entry from ~/.zshrc"; \
			echo "Run: source ~/.zshrc"; \
		else \
			echo "No git-switch PATH entry found in ~/.zshrc"; \
		fi; \
	elif [ -f "$$HOME/.bashrc" ]; then \
		if grep -Fq "$$PATTERN" "$$HOME/.bashrc"; then \
			grep -Fv "$$PATTERN" "$$HOME/.bashrc" > "$$HOME/.bashrc.tmp" && mv "$$HOME/.bashrc.tmp" "$$HOME/.bashrc"; \
			echo "Removed git-switch PATH entry from ~/.bashrc"; \
			echo "Run: source ~/.bashrc"; \
		else \
			echo "No git-switch PATH entry found in ~/.bashrc"; \
		fi; \
	elif [ -f "$$HOME/.config/fish/config.fish" ]; then \
		echo "Run: set -e fish_user_paths[(contains -i $(BINDIR) $$fish_user_paths)]"; \
	else \
		echo "Remove this from your shell profile:"; \
		echo '  export PATH="$$HOME/.local/bin:$$PATH"'; \
	fi

uninstall-completion:
	@rm -f $(ZSH_COMP_DIR)/_git-switch
	@rm -f $(BASH_COMP_DIR)/git-switch
	@rm -f $(FISH_COMP_DIR)/git-switch.fish
	@if [ -f "$$HOME/.zshrc" ]; then \
		if grep -Fq '$(ZSH_FPATH_LINE)' "$$HOME/.zshrc"; then \
			grep -Fv '$(ZSH_FPATH_LINE)' "$$HOME/.zshrc" > "$$HOME/.zshrc.tmp" && mv "$$HOME/.zshrc.tmp" "$$HOME/.zshrc"; \
			echo "Removed zsh completions and fpath entry."; \
			echo "Run: source ~/.zshrc"; \
		else \
			echo "Removed completion files."; \
		fi; \
	else \
		echo "Removed completion files."; \
	fi

clean:
	@rm -f $(BINARY)

test:
	go test ./...
