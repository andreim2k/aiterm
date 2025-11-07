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

	# Start translation in background for multiple options (max 5 most common commands)
	local tmpfile=$(mktemp)
	%s --ai-translate-multiple 5 "$current_buffer" < /dev/null > "$tmpfile" 2>&1 &
	local job=$!

	# Show spinner on a separate line below the prompt (braille characters)
	local spinner=('⠋' '⠙' '⠹' '⠸' '⠼' '⠴' '⠦' '⠧' '⠇' '⠏')
	local i=0

	# Keep the natural language in BUFFER for now (ZLE already displayed it)
	BUFFER="$current_buffer"
	CURSOR=${#BUFFER}

	# Move to next line for spinner (don't overwrite the prompt)
	echo "" >&2
	while kill -0 $job 2>/dev/null; do
		echo -ne "\r\033[K  Translating... ${spinner[i]}" >&2
		i=$(( (i+1) %% 10 ))
		sleep 0.05 || true
	done
	# Clear spinner line
	echo -ne "\r\033[K" >&2
	# Move back up to prompt line
	echo -ne "\033[A" >&2
	# Clear the prompt line with natural language immediately
	echo -ne "\r\033[K" >&2

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
		# Skip empty lines, whitespace-only lines, or lines that are just numbers/punctuation
		# Also ensure line has at least one alphanumeric character
		if [ -n "$line" ] && echo "$line" | grep -qE '[[:alnum:]]' && ! echo "$line" | grep -qE '^[0-9]+[.)]?[[:space:]]*$'; then
			options+=("$line")
			# Limit to maximum 5 commands
			if [ ${#options[@]} -ge 5 ]; then
				break
			fi
		fi
	done <<< "$translated"

	# Remove any empty strings that might have slipped through
	# Create new array and rebuild from non-empty elements
	local -a cleaned_options=()
	local idx
	for ((idx=0; idx<${#options[@]}; idx++)); do
		local opt="${options[$idx]}"
		local stripped="${opt// /}"
		# Skip if empty or only whitespace
		if [[ -n "$stripped" ]]; then
			cleaned_options+=("$opt")
		fi
	done
	# Clear and rebuild original array - use 1-based indexing for zsh
	options=()
	for opt in "${cleaned_options[@]}"; do
		options+=("$opt")
	done

	# Handle different cases: 0 options (do nothing), 1 option (use directly), multiple options (show menu)
	if [ ${#options[@]} -eq 0 ]; then
		# No valid commands found - keep original buffer, do nothing
		BUFFER="$current_buffer"
		CURSOR=${#BUFFER}
		zle -R
	elif [ ${#options[@]} -eq 1 ]; then
		# Single option - use it directly (zsh arrays are 1-indexed)
		BUFFER="${options[1]}"
		CURSOR=${#BUFFER}
		zle -R
	elif [ ${#options[@]} -gt 1 ]; then
		local max_options=${#options[@]}

		# Store options and state in global variables for widget access
		# Note: zsh arrays are 1-indexed
		typeset -g _aiterm_options=("${options[@]}")
		typeset -g _aiterm_selected=1
		typeset -g _aiterm_max_options=$max_options
		typeset -g _aiterm_current_buffer="$current_buffer"
		typeset -g _aiterm_selection_done=0
		typeset -g _aiterm_menu_lines=$((max_options + 1))
		
		local SELECTED_COLOR=$'\033[1;32m'
		local NORMAL_COLOR=$'\033[0m'
		local INSTRUCTIONS_COLOR=$'\033[0;36m'
		
		# Save original cursor position BEFORE displaying menu (where user's prompt cursor was)
		# Use \033[s which might support a stack better than DECSC
		echo -ne "\033[s" >&2  # Save original cursor position
		
		# Function to display selection menu - redraw smoothly
		_aiterm_display_menu() {
			local menu_i option_text

			# Move to saved menu start position (first menu line)
			echo -ne "\033[u" >&2
			# Move up one more line to the prompt
			echo -ne "\033[A" >&2
			# Go to beginning and clear to end of screen
			echo -ne "\r\033[J" >&2

			# Redraw prompt with currently selected command
			print -P -n "$PROMPT" >&2
			echo -n "${_aiterm_options[$_aiterm_selected]}" >&2
			echo "" >&2  # Move to next line

			# Display menu options (zsh arrays are 1-indexed)
			for ((menu_i=1; menu_i<=_aiterm_max_options; menu_i++)); do
				option_text="${_aiterm_options[$menu_i]}"
				if [ $menu_i -eq $_aiterm_selected ]; then
					echo "${SELECTED_COLOR}➤ ${option_text}${NORMAL_COLOR}" >&2
				else
					echo "  ${option_text}" >&2
				fi
			done
			# Instructions line - no newline to prevent scroll
			echo -n "${INSTRUCTIONS_COLOR}↑/↓: Navigate  Enter: Select  Esc/C: Cancel${NORMAL_COLOR}" >&2
		}
		
		# Initial display - Update BUFFER but don't use zle -R (we'll manage display ourselves)
		BUFFER="${options[1]}"
		CURSOR=${#BUFFER}

		# Print prompt with first selected command manually (prompt line already cleared)
		print -P -n "$PROMPT" >&2
		echo -n "${options[1]}" >&2
		echo "" >&2  # Move to next line

		# Save menu start position (at first option line)
		echo -ne "\033[s" >&2
		local menu_idx
		for ((menu_idx=1; menu_idx<=max_options; menu_idx++)); do
			option_text="${options[$menu_idx]}"
			if [ $menu_idx -eq 1 ]; then
				echo "${SELECTED_COLOR}➤ ${option_text}${NORMAL_COLOR}" >&2
			else
				echo "  ${option_text}" >&2
			fi
		done
		echo "${INSTRUCTIONS_COLOR}↑/↓: Navigate  Enter: Select  Esc/C: Cancel${NORMAL_COLOR}" >&2
		
		# Wait for selection - use zsh's read -k reading from /dev/tty
		# We'll read keys in a loop and update the display
		while [ $_aiterm_selection_done -eq 0 ]; do
			# Use read -k to read a single key from /dev/tty (non-blocking with timeout)
			read -k 1 -t 0.1 key < /dev/tty 2>/dev/null || {
				# Timeout - continue loop to check selection_done
				continue
			}
			
			case "$key" in
				$'\e')
					# Escape sequence - read next char from /dev/tty
					read -k 1 -t 0.1 key2 < /dev/tty 2>/dev/null || {
						# Just Escape - cancel
						_aiterm_selection_done=1
						BUFFER="$_aiterm_current_buffer"
						CURSOR=${#BUFFER}
						break
					}
					if [ "$key2" = '[' ]; then
						read -k 1 -t 0.1 key3 < /dev/tty 2>/dev/null || break
						case "$key3" in
							'A') # Up arrow
								if [ $_aiterm_selected -gt 1 ]; then
									_aiterm_selected=$((_aiterm_selected - 1))
									# Update BUFFER with currently selected command
									BUFFER="${_aiterm_options[$_aiterm_selected]}"
									CURSOR=${#BUFFER}
									# Update the display (redraws prompt + menu)
									_aiterm_display_menu
								fi
								;;
							'B') # Down arrow
								if [ $_aiterm_selected -lt $_aiterm_max_options ]; then
									_aiterm_selected=$((_aiterm_selected + 1))
									# Update BUFFER with currently selected command
									BUFFER="${_aiterm_options[$_aiterm_selected]}"
									CURSOR=${#BUFFER}
									# Update the display (redraws prompt + menu)
									_aiterm_display_menu
								fi
								;;
						esac
					else
						# Other escape - cancel
						_aiterm_selection_done=1
						BUFFER="$_aiterm_current_buffer"
						CURSOR=${#BUFFER}
						break
					fi
					;;
				$'\n'|$'\r')
					# Enter - accept
					_aiterm_selection_done=1
					BUFFER="${_aiterm_options[$_aiterm_selected]}"
					CURSOR=${#BUFFER}
					break
					;;
				'c'|'C')
					# Cancel
					_aiterm_selection_done=1
					BUFFER="$_aiterm_current_buffer"
					CURSOR=${#BUFFER}
					break
					;;
			esac
		done
		
		# Clear menu - cursor is at end of instructions line
		# Move cursor up to the prompt line (prompt + options + instructions = max_options + 2 lines total)
		local clear_lines=$((_aiterm_max_options + 2))
		echo -ne "\033[${clear_lines}A" >&2
		# Move to beginning of line and clear everything to end of screen
		echo -ne "\r\033[J" >&2
	fi

	# Redraw prompt with zle (BUFFER already has the selected command)
	zle -R
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
