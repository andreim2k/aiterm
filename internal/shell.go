package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/andreim2k/aiterm/logger"
)

// ShellInterface provides an interactive shell with AI command translation
type ShellInterface struct {
	manager *Manager
	shell   string // bash, zsh, fish, etc.
}

// NewShellInterface creates a new shell interface
func NewShellInterface(manager *Manager) *ShellInterface {
	// Detect the user's shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	return &ShellInterface{
		manager: manager,
		shell:   shell,
	}
}

// Start starts the shell interface with AI command translation
func (s *ShellInterface) Start() error {
	// Create a wrapper script that adds AI translation to the real shell
	wrapperDir, err := s.createShellWrapperScript()
	if err != nil {
		return fmt.Errorf("failed to create wrapper script: %w", err)
	}
	defer os.RemoveAll(wrapperDir)

	fmt.Println("AI Shell Mode (aish) - Press Ctrl+X Ctrl+A to translate natural language to commands")
	fmt.Println("Alternative bindings: Ctrl+Space or Alt+Space")
	fmt.Println("This is your real zsh with AI superpowers!")
	fmt.Println()

	// Run the actual shell with our wrapper using ZDOTDIR
	// Ensure shell is interactive so .zshrc loads
	cmd := exec.Command("zsh", "-i")
	cmd.Env = append(os.Environ(), fmt.Sprintf("ZDOTDIR=%s", wrapperDir))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir, _ = os.Getwd()

	err = cmd.Run()
	if err != nil {
		logger.Debug("Shell exited with error: %v", err)
	} else {
		logger.Debug("Shell exited normally")
	}
	return err
}

// createShellWrapperScript creates a script that adds AI translation keybinding to the shell
func (s *ShellInterface) createShellWrapperScript() (string, error) {
	tmpDir, err := os.MkdirTemp("", "aiterm-zsh-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Create wrapper based on shell type
	if strings.Contains(s.shell, "zsh") {
		// Create .zshrc wrapper
		zshrcPath := filepath.Join(tmpDir, ".zshrc")

		// Source original zshrc first, then add our binding
		homeDir, _ := os.UserHomeDir()

		// Get absolute path to aiterm executable
		aitermPath, absErr := filepath.Abs(os.Args[0])
		if absErr != nil {
			aitermPath = os.Args[0]
		}
		// Quote the path for shell safety
		aitermPath = fmt.Sprintf("\"%s\"", aitermPath)
		logger.Debug("Using aiterm path: %s", aitermPath)

		content := fmt.Sprintf(`# Source original zshrc
if [ -f "%s/.zshrc" ]; then
	source "%s/.zshrc"
fi

# Prevent shell from exiting on Ctrl+D
setopt ignore_eof

# AI translation function
ai-translate-command() {
	# Disable job control notifications
	setopt local_options no_notify no_monitor

	local current_buffer="$BUFFER"

	if [ -z "$current_buffer" ]; then
		return
	fi

	# Start translation in background for multiple options
	local tmpfile=$(mktemp)
	%s --ai-translate-multiple 10 "$current_buffer" < /dev/null > "$tmpfile" 2>&1 &
	local job=$!

	# Spinner animation with braille characters
	local spinner=('⠋' '⠙' '⠹' '⠸' '⠼' '⠴' '⠦' '⠧' '⠇' '⠏')
	local i=0
	while kill -0 $job 2>/dev/null; do
		BUFFER="$current_buffer ${spinner[i]}"
		zle -R
		i=$(( (i+1) %% 10 ))
		sleep 0.05 || true
	done

	# Wait for job to complete
	wait $job 2>/dev/null || true

	# Get result
	local translated=$(cat $tmpfile)
	rm $tmpfile

	# Parse multiple options (separated by newlines)
	local -a options
	local index=1
	while IFS= read -r line; do
		if [ -n "$line" ]; then
			options+=("$line")
		fi
	done <<< "$translated"

	# If we have multiple options, show selection interface
	if [ ${#options[@]} -gt 1 ]; then
		# Show options numbered for easy selection
		local max_options=${#options[@]}
		local i
		
		# Clear and show options
		echo ""
		for ((i=1; i<=max_options; i++)); do
			echo "  $i) ${options[$((i-1))]}"
		done
		echo "  0) Cancel (restore original text)"
		echo ""
		echo -n "Select option (0-$max_options, default: 1): "
		
		# Read selection - use vared for better ZLE integration
		local selection=""
		# Temporarily disable ZLE to read input
		zle -I
		read -r selection < /dev/tty 2>/dev/null || read -r selection
		
		# Parse selection
		local selected=${selection:-1}
		if [ "$selected" = "0" ] || [ -z "$selected" ]; then
			# Cancel or empty - restore original buffer
			BUFFER="$current_buffer"
			CURSOR=${#BUFFER}
		elif [ "$selected" -ge 1 ] && [ "$selected" -le $max_options ]; then
			# Valid selection
			BUFFER="${options[$((selected-1))]}"
			CURSOR=${#BUFFER}
		else
			# Invalid selection - use first option
			BUFFER="${options[0]}"
			CURSOR=${#BUFFER}
		fi
		
		# Clear the selection prompt
		local lines_to_clear=$((max_options + 4))
		echo -ne "\033[${lines_to_clear}A\033[J"
	else
		# Single option - replace buffer directly
		if [ -n "$translated" ]; then
			BUFFER="$translated"
			CURSOR=${#BUFFER}
		else
			# Restore original if translation failed
			BUFFER="$current_buffer"
			CURSOR=${#BUFFER}
		fi
	fi

	# Redraw
	zle reset-prompt
	return 0
}

# Register as a ZLE widget
zle -N ai-translate-command

# Function to set up key bindings - must be called when ZLE is active
setup-aiterm-bindings() {
	# Only set up bindings once to avoid duplicate output
	[ -n "$_aiterm_bindings_set" ] && return
	_aiterm_bindings_set=1
	
	# Ensure keymaps are available
	zmodload zsh/terminfo 2>/dev/null || true
	
	# Check if widget exists
	if ! zle -l | grep -q "ai-translate-command"; then
		return 1
	fi
	
	# Bind to Ctrl+X then Ctrl+A (most reliable, works in most terminals)
	bindkey '^X^A' ai-translate-command 2>/dev/null || true
	
	# Bind to Ctrl+Space (^@ is the control sequence for Ctrl+Space)
	# Note: This may not work in all terminals
	bindkey -M emacs '^@' ai-translate-command 2>/dev/null || true
	bindkey -M viins '^@' ai-translate-command 2>/dev/null || true
	bindkey -M vicmd '^@' ai-translate-command 2>/dev/null || true
	bindkey '^@' ai-translate-command 2>/dev/null || true
	
	# Alternative binding: Alt+Space (more compatible)
	bindkey '\e ' ai-translate-command 2>/dev/null || true
}

# Use zsh's zle-line-init hook to set bindings when line editor initializes
# This ensures bindings are set after user's zshrc loads and persist
# Check if user already has zle-line-init and preserve it
if typeset -f zle-line-init > /dev/null 2>&1; then
	# User has zle-line-init, wrap it properly
	_aiterm_original_zle_line_init=$(functions zle-line-init)
	_aiterm_zle_line_init_wrapper() {
		eval "$_aiterm_original_zle_line_init"
		setup-aiterm-bindings
	}
	zle -N zle-line-init _aiterm_zle_line_init_wrapper
else
	# No existing zle-line-init, create our own
	zle-line-init() {
		setup-aiterm-bindings
	}
	zle -N zle-line-init
fi

# Also set up bindings immediately (for first prompt)
setup-aiterm-bindings

# Use precmd hook as backup to ensure bindings are set (only once)
autoload -Uz add-zsh-hook
aiterm-setup-bindings-hook() {
	setup-aiterm-bindings
	# Remove hook after first run
	add-zsh-hook -d precmd aiterm-setup-bindings-hook
}
add-zsh-hook precmd aiterm-setup-bindings-hook

# Note: Ctrl+Tab is often intercepted by terminal emulators and may not work
# Uncomment the following line if your terminal supports it:
# bindkey '^[^I' ai-translate-command
`, homeDir, homeDir, aitermPath)

		err = os.WriteFile(zshrcPath, []byte(content), 0600)
		if err != nil {
			return "", fmt.Errorf("failed to write zshrc: %w", err)
		}

		logger.Debug("Created wrapper at %s", tmpDir)

		// Also write the content to a persistent file for debugging
		debugPath := "/tmp/aiterm-debug-wrapper.zsh"
		if writeErr := os.WriteFile(debugPath, []byte(content), 0644); writeErr != nil {
			logger.Debug("Failed to write debug wrapper: %v", writeErr)
		} else {
			logger.Debug("Debug wrapper written to: %s", debugPath)
		}

		return tmpDir, nil
	}

	// TODO: Add bash support
	return "", fmt.Errorf("shell wrapper not implemented for %s yet", s.shell)
}

func TranslateNaturalLanguage(mgr *Manager, naturalLanguage string) (string, error) {
	// Build AI prompt for command translation
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		shellPath = "/bin/bash"
	}

	cwd, _ := os.Getwd()
	systemPrompt := fmt.Sprintf(`You are a shell command translator. Convert natural language to shell commands.

Operating System: %s
Shell: %s
Current Directory: %s

Rules:
1. Output ONLY a single shell command, nothing else
2. No explanations, no comments, no markdown
3. Command should be safe and follow best practices

Examples:
Input: "list all files"
Output: ls -la

Input: "find python files"
Output: find . -name "*.py"

Respond with ONLY the command.`, mgr.OS, shellPath, cwd)

	userPrompt := fmt.Sprintf("Translate: %s", naturalLanguage)

	// Create chat messages
	messages := []ChatMessage{
		{Content: systemPrompt, FromUser: false},
		{Content: userPrompt, FromUser: true},
	}

	// Call AI
	ctx := context.Background()
	response, err := mgr.AiClient.GetResponseFromChatMessages(ctx, messages, mgr.GetModel())
	if err != nil {
		return "", fmt.Errorf("failed to get AI response: %w", err)
	}

	// Clean up response
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```bash")
	response = strings.TrimPrefix(response, "```sh")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	return response, nil
}
