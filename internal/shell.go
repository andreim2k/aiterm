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
	while IFS= read -r line; do
		# Remove leading numbers with various formats: "1. ", "1) ", "1)", "1.", etc.
		line=$(echo "$line" | sed -E 's/^[[:space:]]*[0-9]+[.)][[:space:]]*//')
		# Trim whitespace
		line=$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
		# Skip empty lines or lines that are just numbers/punctuation
		if [ -n "$line" ] && ! echo "$line" | grep -qE '^[0-9]+[.)]?[[:space:]]*$'; then
			options+=("$line")
		fi
	done <<< "$translated"

	# If we have multiple options, show selection interface with arrow key navigation
	if [ ${#options[@]} -gt 1 ]; then
		local max_options=${#options[@]}
		
		# Store options and state in global variables for widget access
		typeset -g _aiterm_options=("${options[@]}")
		typeset -g _aiterm_selected=1
		typeset -g _aiterm_max_options=$max_options
		typeset -g _aiterm_current_buffer="$current_buffer"
		typeset -g _aiterm_selection_done=0
		typeset -g _aiterm_menu_lines=$((max_options + 1))
		
		local SELECTED_COLOR=$'\033[1;32m'
		local NORMAL_COLOR=$'\033[0m'
		local INSTRUCTIONS_COLOR=$'\033[0;36m'
		
		# Function to display selection menu
		_aiterm_display_menu() {
			# Temporarily restore terminal to cooked mode for display
			# Use the global tty_fd if available, otherwise use /dev/tty directly
			local tty_to_use=${_aiterm_tty_fd:-/dev/tty}
			if [[ "$tty_to_use" =~ ^[0-9]+$ ]]; then
				# It's a file descriptor number
				local display_stty=$(stty -g <&$tty_to_use 2>/dev/null)
				stty cooked echo <&$tty_to_use 2>/dev/null || true
			else
				# It's a path, open it
				exec {local_fd}<>"$tty_to_use" 2>/dev/null || return
				local display_stty=$(stty -g <&$local_fd 2>/dev/null)
				stty cooked echo <&$local_fd 2>/dev/null || true
			fi
			
			local i idx option_text
			# Move cursor up to start of menu (max_options + 1 for instructions line)
			printf "\033[${_aiterm_menu_lines}A" >&2
			# Display each option
			for ((i=1; i<=_aiterm_max_options; i++)); do
				idx=$((i-1))
				option_text="${_aiterm_options[$idx]}"
				if [ $i -eq $_aiterm_selected ]; then
					printf "\033[2K${SELECTED_COLOR}➤ ${option_text}${NORMAL_COLOR}\r\n" >&2
				else
					printf "\033[2K  ${option_text}\r\n" >&2
				fi
			done
			printf "\033[2K${INSTRUCTIONS_COLOR}↑/↓: Navigate  Enter: Select  Esc/C: Cancel${NORMAL_COLOR}\r" >&2
			
			# Restore raw mode for key reading
			if [[ "$tty_to_use" =~ ^[0-9]+$ ]]; then
				stty -icanon -echo min 0 time 0 <&$tty_to_use 2>/dev/null || true
			else
				stty -icanon -echo min 0 time 0 <&$local_fd 2>/dev/null || true
				exec {local_fd}<&-
			fi
		}
		
		
		# Initial display
		echo "" >&2
		for ((i=1; i<=max_options; i++)); do
			idx=$((i-1))
			option_text="${options[$idx]}"
			if [ $i -eq 1 ]; then
				echo "${SELECTED_COLOR}➤ ${option_text}${NORMAL_COLOR}" >&2
			else
				echo "  ${option_text}" >&2
			fi
		done
		echo "${INSTRUCTIONS_COLOR}↑/↓: Navigate  Enter: Select  Esc/C: Cancel${NORMAL_COLOR}" >&2
		
		# Wait for selection by reading keys directly from terminal
		# We need to read keys without zle interfering
		local tty_fd
		if ! exec {tty_fd}<>/dev/tty 2>/dev/null; then
			# Fallback: use first option if we can't open terminal
			BUFFER="${options[0]}"
			CURSOR=${#BUFFER}
			zle reset-prompt
			return 0
		fi
		
		# Store tty_fd globally so display function can access it
		typeset -g _aiterm_tty_fd=$tty_fd
		
		# Save terminal state and set raw mode for key reading
		# Use -icanon to disable canonical mode, but keep output working
		local old_stty=$(stty -g <&$tty_fd 2>/dev/null)
		# Set raw mode: disable echo and canonical mode, but allow output
		stty -icanon -echo min 0 time 0 <&$tty_fd 2>/dev/null || {
			# If stty fails, close fd and use first option
			exec {tty_fd}<&-
			BUFFER="${options[0]}"
			CURSOR=${#BUFFER}
			zle reset-prompt
			return 0
		}
		
		# Read keys until selection is done
		while [ $_aiterm_selection_done -eq 0 ]; do
			# Read a single byte (dd will block until data is available)
			local key=$(dd bs=1 count=1 <&$tty_fd 2>/dev/null)
			[ -z "$key" ] && continue
			
			case "$key" in
				$'\e')
					# Escape sequence - read next char
					local key2=$(dd bs=1 count=1 <&$tty_fd 2>/dev/null)
					if [ "$key2" = '[' ]; then
						local key3=$(dd bs=1 count=1 <&$tty_fd 2>/dev/null)
						case "$key3" in
							'A') # Up arrow
								if [ $_aiterm_selected -gt 1 ]; then
									_aiterm_selected=$((_aiterm_selected - 1))
									_aiterm_display_menu
								fi
								;;
							'B') # Down arrow
								if [ $_aiterm_selected -lt $_aiterm_max_options ]; then
									_aiterm_selected=$((_aiterm_selected + 1))
									_aiterm_display_menu
								fi
								;;
						esac
					else
						# Just Escape - cancel
						_aiterm_selection_done=1
						BUFFER="$_aiterm_current_buffer"
						CURSOR=${#BUFFER}
						break
					fi
					;;
				$'\n'|$'\r')
					# Enter - accept selection
					_aiterm_selection_done=1
					local idx=$((_aiterm_selected-1))
					BUFFER="${_aiterm_options[$idx]}"
					CURSOR=${#BUFFER}
					break
					;;
				'c'|'C')
					# Cancel with 'c'
					_aiterm_selection_done=1
					BUFFER="$_aiterm_current_buffer"
					CURSOR=${#BUFFER}
					break
					;;
				$'\x03')
					# Ctrl+C - cancel
					_aiterm_selection_done=1
					BUFFER="$_aiterm_current_buffer"
					CURSOR=${#BUFFER}
					break
					;;
			esac
		done
		
		# Restore terminal state before clearing menu
		stty "$old_stty" <&$tty_fd 2>/dev/null || true
		exec {tty_fd}<&-
		
		# Clear menu from screen
		echo -ne "\033[${_aiterm_menu_lines}A\033[J" >&2
		
		# Clean up global variables
		unset _aiterm_options _aiterm_selected _aiterm_max_options _aiterm_current_buffer _aiterm_selection_done _aiterm_menu_lines _aiterm_tty_fd
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
zle -N ai-translate-command >/dev/null 2>&1

# Function to set up key bindings - must be called when ZLE is active
setup-aiterm-bindings() {
	# Only set up bindings once to avoid duplicate output
	[ -n "$_aiterm_bindings_set" ] && return
	_aiterm_bindings_set=1
	
	# Ensure keymaps are available (suppress all output)
	zmodload zsh/terminfo >/dev/null 2>&1 || true
	
	# Check if widget exists (suppress grep output)
	if ! zle -l 2>/dev/null | grep -q "ai-translate-command" 2>/dev/null; then
		return 1
	fi
	
	# Bind to Ctrl+X then Ctrl+A (most reliable, works in most terminals)
	# Suppress all output including stderr
	bindkey '^X^A' ai-translate-command >/dev/null 2>&1 || true
	
	# Bind to Ctrl+Space (^@ is the control sequence for Ctrl+Space)
	# Note: This may not work in all terminals
	bindkey -M emacs '^@' ai-translate-command >/dev/null 2>&1 || true
	bindkey -M viins '^@' ai-translate-command >/dev/null 2>&1 || true
	bindkey -M vicmd '^@' ai-translate-command >/dev/null 2>&1 || true
	bindkey '^@' ai-translate-command >/dev/null 2>&1 || true
	
	# Alternative binding: Alt+Space (more compatible)
	bindkey '\e ' ai-translate-command >/dev/null 2>&1 || true
}

# Use zsh's zle-line-init hook to set bindings when line editor initializes
# This ensures bindings are set after user's zshrc loads and persist
# Check if user already has zle-line-init and preserve it
if typeset -f zle-line-init >/dev/null 2>&1; then
	# User has zle-line-init, wrap it properly
	_aiterm_original_zle_line_init=$(functions zle-line-init)
	_aiterm_zle_line_init_wrapper() {
		eval "$_aiterm_original_zle_line_init" >/dev/null 2>&1
		setup-aiterm-bindings >/dev/null 2>&1
	}
	zle -N zle-line-init _aiterm_zle_line_init_wrapper >/dev/null 2>&1
else
	# No existing zle-line-init, create our own
	zle-line-init() {
		setup-aiterm-bindings >/dev/null 2>&1
	}
	zle -N zle-line-init >/dev/null 2>&1
fi

# Also set up bindings immediately (for first prompt) - suppress output
setup-aiterm-bindings >/dev/null 2>&1

# Use precmd hook as backup to ensure bindings are set (only once)
autoload -Uz add-zsh-hook
aiterm-setup-bindings-hook() {
	setup-aiterm-bindings >/dev/null 2>&1
	# Remove hook after first run
	add-zsh-hook -d precmd aiterm-setup-bindings-hook >/dev/null 2>&1
}
add-zsh-hook precmd aiterm-setup-bindings-hook >/dev/null 2>&1

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
