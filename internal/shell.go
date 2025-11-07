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

	fmt.Println("AI Shell Mode (aish) - Press Ctrl+Space to translate natural language to commands")
	fmt.Println("This is your real zsh with AI superpowers!")
	fmt.Println()

	// Run the actual shell with our wrapper using ZDOTDIR
	cmd := exec.Command("zsh")
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
	# Debug output to verify function is called
	# echo "AI translation function called" >&2
	
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
		# Show options with selection interface
		local selected=1
		local max_options=${#options[@]}
		
		# Add Cancel option
		options+=("Cancel (restore original text)")
		max_options=${#options[@]}
		
		# ANSI color codes for better visual feedback
		local SELECTED_COLOR=$'\033[1;32m'  # Bold green
		local CANCEL_COLOR=$'\033[0;31m'    # Red
		local NORMAL_COLOR=$'\033[0m'       # Reset to normal
		local INSTRUCTIONS_COLOR=$'\033[0;36m'  # Cyan
		local SELECTED_MARKER="➤"
		local UNSELECTED_MARKER=" "
		
		# Function to clear and display options with selection marker
		display_options() {
			local i
			# Move cursor up by max_options lines and clear from there
			echo -ne "\033[${max_options}A\033[J"
			for ((i=1; i<=max_options; i++)); do
				if [ $i -eq $selected ]; then
					if [ $i -eq $max_options ]; then
						# Cancel option - show in red
						echo "${CANCEL_COLOR}${SELECTED_MARKER} ${options[$((i-1))]}${NORMAL_COLOR}"
					else
						echo "${SELECTED_COLOR}${SELECTED_MARKER} ${options[$((i-1))]}${NORMAL_COLOR}"  # Selected option (bold green)
					fi
				else
					if [ $i -eq $max_options ]; then
						# Cancel option - show in red
						echo "${CANCEL_COLOR}${UNSELECTED_MARKER} ${options[$((i-1))]}${NORMAL_COLOR}"
					else
						echo "${UNSELECTED_MARKER} ${options[$((i-1))]}"  # Unselected option
					fi
				fi
			done
			# Show instructions
			echo "${INSTRUCTIONS_COLOR}↑/↓: Navigate  Enter: Select  Number: Direct select  Q/Esc/Ctrl+C: Cancel${NORMAL_COLOR}"
		}
		
		# Show options initially (with extra lines for spacing and instructions)
		echo ""
		for ((i=1; i<=max_options; i++)); do
			if [ $i -eq $selected ]; then
				if [ $i -eq $max_options ]; then
					# Cancel option - show in red
					echo "${CANCEL_COLOR}${SELECTED_MARKER} ${options[$((i-1))]}${NORMAL_COLOR}"
				else
					echo "${SELECTED_COLOR}${SELECTED_MARKER} ${options[$((i-1))]}${NORMAL_COLOR}"  # Selected option (bold green)
				fi
			else
				if [ $i -eq $max_options ]; then
					# Cancel option - show in red
					echo "${CANCEL_COLOR}${UNSELECTED_MARKER} ${options[$((i-1))]}${NORMAL_COLOR}"
				else
					echo "${UNSELECTED_MARKER} ${options[$((i-1))]}"  # Unselected option
				fi
			done
		done
		# Show instructions
		echo "${INSTRUCTIONS_COLOR}↑/↓: Navigate  Enter: Select  Number: Direct select  Q/Esc/Ctrl+C: Cancel${NORMAL_COLOR}"
		
		# Selection loop
		while true; do
			# Read input character
			local key
			read -k 1 key
			
			case "$key" in
				$'\x1b')  # ESC sequence (arrow keys start with ESC)
					# Try to read the rest of the escape sequence
					local key2 key3
					read -k 1 -t 0.01 key2 2>/dev/null || key2=""
					if [ -n "$key2" ]; then
						read -k 1 -t 0.01 key3 2>/dev/null || key3=""
						if [ -n "$key3" ]; then
							case "$key2$key3" in
								'[A'|'OA')  # Up arrow (both ESC[A and ESCOA)
									if [ $selected -gt 1 ]; then
										selected=$((selected - 1))
										display_options
									fi
									;;
								'[B'|'OB')  # Down arrow (both ESC[B and ESCOB)
									if [ $selected -lt $max_options ]; then
										selected=$((selected + 1))
										display_options
									fi
									;;
							esac
						fi
					else
						# ESC key alone - cancel and restore original text
						BUFFER="$current_buffer"
						CURSOR=${#BUFFER}
						# Clear all options plus the extra lines
						echo -ne "\033[$((max_options + 2))A\033[J"
						break
					fi
					;;
				'')  # Enter key
					# Check if Cancel was selected
					if [ $selected -eq $max_options ]; then
						# Cancel - restore original buffer
						BUFFER="$current_buffer"
						CURSOR=${#BUFFER}
					else
						# Use selected option
						BUFFER="${options[$((selected-1))]}"
						CURSOR=${#BUFFER}
					fi
					# Clear all options plus the extra lines
					echo -ne "\033[$((max_options + 2))A\033[J"
					break
					;;
				'q'|'Q')  # Q key - cancel
					# Restore original buffer
					BUFFER="$current_buffer"
					CURSOR=${#BUFFER}
					# Clear all options plus the extra lines
					echo -ne "\033[$((max_options + 2))A\033[J"
					break
					;;
				$'\x03')  # Ctrl+C - cancel
					# Restore original buffer
					BUFFER="$current_buffer"
					CURSOR=${#BUFFER}
					# Clear all options plus the extra lines
					echo -ne "\033[$((max_options + 2))A\033[J"
					break
					;;
				[1-9])  # Number keys
					# Direct selection by number (if valid)
					local num=$(printf "%d" "'$key")
					num=$((num - 48))  # Convert ASCII to number
					if [ $num -lt $max_options ]; then
						# Valid option (not Cancel)
						selected=$num
						BUFFER="${options[$((selected-1))]}"
						CURSOR=${#BUFFER}
						# Clear all options plus the extra lines
						echo -ne "\033[$((max_options + 2))A\033[J"
						break
					elif [ $num -eq $max_options ]; then
						# Cancel option
						BUFFER="$current_buffer"
						CURSOR=${#BUFFER}
						# Clear all options plus the extra lines
						echo -ne "\033[$((max_options + 2))A\033[J"
						break
					fi
					;;
			esac
		done
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
	echo "Registering ai-translate-command as ZLE widget" >&2
	zle -N ai-translate-command

# Debug: Show that the function is loaded
echo "AI Translate function loaded" >&2

# Ensure keymaps are available
zmodload zsh/terminfo 2>/dev/null || true

# Bind to Ctrl+Space (^@ is the control sequence for Ctrl+Space)
	# This is the recommended primary binding
	echo "Binding Ctrl+Space to ai-translate-command" >&2
	bindkey -M emacs '^@' ai-translate-command
	bindkey -M viins '^@' ai-translate-command
	bindkey -M vicmd '^@' ai-translate-command

	# Also try alternative bindings for better compatibility
	bindkey '^@' ai-translate-command

	# Alternative binding for tmux compatibility: Ctrl+X then Ctrl+A
	echo "Binding Ctrl+X Ctrl+A to ai-translate-command" >&2
	bindkey '^X^A' ai-translate-command

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
