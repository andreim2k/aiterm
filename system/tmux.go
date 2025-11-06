package system

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/andreim2k/aiterm/logger"
)

// TmuxCreateNewPane creates a new vertical split pane in the specified window and returns its ID
func TmuxCreateNewPane(target string) (string, error) {
	cmd := exec.Command("tmux", "split-window", "-d", "-v", "-t", target, "-P", "-F", "#{pane_id}")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to create tmux pane: %v, stderr: %s", err, stderr.String())
		return "", err
	}

	paneId := strings.TrimSpace(stdout.String())
	return paneId, nil
}

// TmuxPanesDetails gets details for all panes in a target window
var TmuxPanesDetails = func(target string) ([]TmuxPaneDetails, error) {
	cmd := exec.Command("tmux", "list-panes", "-t", target, "-F", "#{pane_id},#{pane_active},#{pane_pid},#{pane_current_command},#{history_size},#{history_limit}")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to get tmux pane details for target %s %v, stderr: %s", target, err, stderr.String())
		return nil, err
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return nil, fmt.Errorf("no pane details found for target %s", target)
	}

	lines := strings.Split(output, "\n")
	paneDetails := make([]TmuxPaneDetails, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ",", 6)
		if len(parts) < 5 {
			logger.Error("Invalid pane details format for line: %s", line)
			continue
		}

		id := parts[0]

		// If target starts with '%', it's a pane ID, so only include the matching pane
		if strings.HasPrefix(target, "%") && id != target {
			continue
		}

		active, _ := strconv.Atoi(parts[1])
		pid, _ := strconv.Atoi(parts[2])
		historySize, _ := strconv.Atoi(parts[4])
		historyLimit, _ := strconv.Atoi(parts[5])
		currentCommandArgs := GetProcessArgs(pid)
		isSubShell := IsSubShell(parts[3])

		paneDetail := TmuxPaneDetails{
			Id:                 id,
			IsActive:           active,
			CurrentPid:         pid,
			CurrentCommand:     parts[3],
			CurrentCommandArgs: currentCommandArgs,
			HistorySize:        historySize,
			HistoryLimit:       historyLimit,
			IsSubShell:         isSubShell,
		}

		paneDetails = append(paneDetails, paneDetail)
	}

	return paneDetails, nil
}

// TmuxCapturePane gets the content of a specific pane by ID
var TmuxCapturePane = func(paneId string, maxLines int) (string, error) {
	cmd := exec.Command("tmux", "capture-pane", "-p", "-t", paneId, "-S", fmt.Sprintf("-%d", maxLines))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to capture pane content from %s: %v, stderr: %s", paneId, err, stderr.String())
		return "", err
	}

	content := strings.TrimSpace(stdout.String())
	return content, nil
}

// Return current tmux window target with session id and window id
func TmuxCurrentWindowTarget() (string, error) {
	paneId, err := TmuxCurrentPaneId()
	if err != nil {
		return "", err
	}

	cmd := exec.Command("tmux", "list-panes", "-t", paneId, "-F", "#{session_id}:#{window_index}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get window target: %w", err)
	}

	target := strings.TrimSpace(string(output))
	if target == "" {
		return "", fmt.Errorf("empty window target returned")
	}

	if idx := strings.Index(target, "\n"); idx != -1 {
		target = target[:idx]
	}

	return target, nil
}

var TmuxCurrentPaneId = func() (string, error) {
	tmuxPane := os.Getenv("TMUX_PANE")
	if tmuxPane == "" {
		return "", fmt.Errorf("TMUX_PANE environment variable not set")
	}

	return tmuxPane, nil
}

// CreateTmuxSession creates a new tmux session and returns the new pane id
func TmuxCreateSession() (string, error) {
	cmd := exec.Command("tmux", "new-session", "-d", "-P", "-F", "#{pane_id}")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		logger.Error("Failed to create tmux session: %v, stderr: %s", err, stderr.String())
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// AttachToTmuxSession attaches to an existing tmux session
func TmuxAttachSession(paneId string) error {
	cmd := exec.Command("tmux", "attach-session", "-t", paneId)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Error("Failed to attach to tmux session: %v", err)
		return err
	}
	return nil
}

// TmuxExecSession runs tmux new-session with proper terminal handling
// This avoids creating background jobs in the parent shell
func TmuxExecSession(args []string) error {
	// Find the tmux binary path
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("failed to find tmux binary: %w", err)
	}

	// Create command with proper I/O handling
	cmd := exec.Command(tmuxPath, args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run tmux and wait for it to complete
	return cmd.Run()
}

func TmuxClearPane(paneId string) error {
	paneDetails, err := TmuxPanesDetails(paneId)
	if err != nil {
		logger.Error("Failed to get pane details for %s: %v", paneId, err)
		return err
	}

	if len(paneDetails) == 0 {
		return fmt.Errorf("no pane details found for pane %s", paneId)
	}

	cmd := exec.Command("tmux", "split-window", "-vp", "100", "-t", paneId)
	if err := cmd.Run(); err != nil {
		logger.Error("Failed to split window for pane %s: %v", paneId, err)
		return err
	}

	cmd = exec.Command("tmux", "clear-vistory", "-t", paneId)
	if err := cmd.Run(); err != nil {
		logger.Error("Failed to clear history for pane %s: %v", paneId, err)
		return err
	}

	cmd = exec.Command("tmux", "kill-pane")
	if err := cmd.Run(); err != nil {
		logger.Error("Failed to kill temporary pane: %v", err)
		return err
	}

	logger.Debug("Successfully cleared pane %s", paneId)
	return nil
}

// TmuxSetPaneTitle sets the title of a tmux pane
func TmuxSetPaneTitle(paneId string, title string) error {
	cmd := exec.Command("tmux", "select-pane", "-t", paneId, "-T", title)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to set pane title for %s: %v, stderr: %s", paneId, err, stderr.String())
		return err
	}

	logger.Debug("Set pane title for %s to: %s", paneId, title)
	return nil
}

// TmuxKillPane kills a specific tmux pane
func TmuxKillPane(paneId string) error {
	cmd := exec.Command("tmux", "kill-pane", "-t", paneId)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to kill pane %s: %v, stderr: %s", paneId, err, stderr.String())
		return err
	}

	logger.Debug("Killed pane %s", paneId)
	return nil
}

// TmuxSwitchToOtherPane switches between the current pane and another specified pane
func TmuxSwitchToOtherPane(chatPaneId, execPaneId string) error {
	// Get current pane ID
	currentPaneId, err := TmuxCurrentPaneId()
	if err != nil {
		logger.Error("Failed to get current pane ID: %v", err)
		return err
	}

	// Determine which pane to switch to
	var targetPane string
	if currentPaneId == chatPaneId {
		targetPane = execPaneId
	} else {
		targetPane = chatPaneId
	}

	// Switch to the target pane
	cmd := exec.Command("tmux", "select-pane", "-t", targetPane)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		logger.Error("Failed to switch to pane %s: %v, stderr: %s", targetPane, err, stderr.String())
		return err
	}

	logger.Debug("Switched from pane %s to pane %s", currentPaneId, targetPane)
	return nil
}

// TmuxSetupPaneSwitchBinding sets up a tmux key binding for Shift+Tab to switch between panes
func TmuxSetupPaneSwitchBinding(chatPaneId, execPaneId string) error {
	// Create a tmux key binding for Shift+Tab (BTab in tmux notation)
	// The binding will run a command that determines current pane and switches to the other
	switchCmd := fmt.Sprintf(
		"if-shell '[ #{pane_id} = %s ]' 'select-pane -t %s' 'select-pane -t %s'",
		chatPaneId, execPaneId, chatPaneId,
	)

	cmd := exec.Command("tmux", "bind-key", "-n", "BTab", switchCmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to set up Shift+Tab binding: %v, stderr: %s", err, stderr.String())
		return err
	}

	logger.Debug("Set up Shift+Tab binding to switch between panes %s and %s", chatPaneId, execPaneId)
	return nil
}

// TmuxSetupPaneToggleBinding sets up Shift+Down arrow to toggle chat pane collapse/expand
func TmuxSetupPaneToggleBinding(chatPaneId, execPaneId string) error {
	// First unbind any existing S-Down bindings to avoid conflicts
	// Unbind from root table (our custom binding from previous session)
	unbindCmd := exec.Command("tmux", "unbind-key", "-n", "S-Down")
	_ = unbindCmd.Run() // Ignore error if no binding exists

	// Unbind from prefix table (default tmux binding)
	unbindCmd = exec.Command("tmux", "unbind-key", "-T", "prefix", "S-Down")
	_ = unbindCmd.Run() // Ignore error if no binding exists

	// Use pure tmux commands - if chat pane is small, expand it; otherwise collapse it
	// This avoids external process calls that create background job notifications
	toggleCmd := fmt.Sprintf(
		"if-shell '[ $(tmux display-message -t %s -p \"#{pane_height}\") -le 2 ]' "+
			"'select-layout even-vertical ; select-pane -t %s' "+
			"'resize-pane -t %s -y 1 ; select-pane -t %s'",
		chatPaneId,
		chatPaneId,
		chatPaneId, execPaneId,
	)

	// Shift+Down arrow is represented as S-Down in tmux
	cmd := exec.Command("tmux", "bind-key", "-n", "S-Down", toggleCmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to set up Shift+Down arrow binding: %v, stderr: %s", err, stderr.String())
		return err
	}

	logger.Debug("Set up Shift+Down arrow binding to toggle pane collapse for %s and %s", chatPaneId, execPaneId)
	return nil
}

// TmuxSetupStyling sets up tmux color scheme (gray borders, blue status bar)
func TmuxSetupStyling() error {
	commands := [][]string{
		// Enable mouse support
		{"tmux", "set", "-g", "mouse", "on"},
		// Set pane borders to gray
		{"tmux", "set", "-g", "pane-border-style", "fg=colour240"},
		{"tmux", "set", "-g", "pane-active-border-style", "fg=colour245"},
		// Set status bar to blue
		{"tmux", "set", "-g", "status-style", "bg=colour25,fg=white"},
	}

	for _, cmdArgs := range commands {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			logger.Error("Failed to set tmux style %v: %v, stderr: %s", cmdArgs, err, stderr.String())
			return err
		}
	}

	logger.Debug("Set up tmux styling: gray borders, blue status bar")
	return nil
}

// TmuxUpdateStatusBar updates the tmux status bar with model information
func TmuxUpdateStatusBar(modelName, provider string) error {
	statusText := fmt.Sprintf("aiterm | model: %s (%s)", modelName, provider)

	cmd := exec.Command("tmux", "set", "-g", "status-right", statusText)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to update status bar: %v, stderr: %s", err, stderr.String())
		return err
	}

	logger.Debug("Updated status bar with model: %s (%s)", modelName, provider)
	return nil
}

// TmuxGetPaneHeight gets the height of a specific pane
func TmuxGetPaneHeight(paneId string) (int, error) {
	cmd := exec.Command("tmux", "display-message", "-t", paneId, "-p", "#{pane_height}")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to get pane height for %s: %v, stderr: %s", paneId, err, stderr.String())
		return 0, err
	}

	height := strings.TrimSpace(stdout.String())
	heightInt, err := strconv.Atoi(height)
	if err != nil {
		logger.Error("Failed to parse pane height '%s': %v", height, err)
		return 0, err
	}

	return heightInt, nil
}

// TmuxResizePane resizes a pane to a specific height
func TmuxResizePane(paneId string, height int) error {
	cmd := exec.Command("tmux", "resize-pane", "-t", paneId, "-y", strconv.Itoa(height))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to resize pane %s to height %d: %v, stderr: %s", paneId, height, err, stderr.String())
		return err
	}

	logger.Debug("Resized pane %s to height %d", paneId, height)
	return nil
}

// TmuxTogglePaneCollapse toggles between collapsed (1 line) and expanded state
func TmuxTogglePaneCollapse(chatPaneId, execPaneId string) error {
	// Get current height of chat pane
	chatHeight, err := TmuxGetPaneHeight(chatPaneId)
	if err != nil {
		return err
	}

	// If chat pane is very small (<=2 lines), expand it
	if chatHeight <= 2 {
		// Expand chat pane to 50% of window
		err := TmuxResizePane(chatPaneId, 0)
		if err != nil {
			return err
		}
		// Make panes equal size
		cmd := exec.Command("tmux", "select-layout", "even-vertical")
		if err := cmd.Run(); err != nil {
			logger.Error("Failed to set even layout: %v", err)
			return err
		}
		// Focus chat pane
		cmd = exec.Command("tmux", "select-pane", "-t", chatPaneId)
		if err := cmd.Run(); err != nil {
			return err
		}
		logger.Debug("Expanded chat pane")
	} else {
		// Collapse chat pane to 1 line
		err := TmuxResizePane(chatPaneId, 1)
		if err != nil {
			return err
		}
		// Focus exec pane
		cmd := exec.Command("tmux", "select-pane", "-t", execPaneId)
		if err := cmd.Run(); err != nil {
			return err
		}
		logger.Debug("Collapsed chat pane")
	}

	return nil
}

// TmuxSetupPaneResizeBindings sets up Shift+Up and Shift+Down to resize the active pane
func TmuxSetupPaneResizeBindings() error {
	// Unbind any existing S-Up and S-Down bindings
	unbindUp := exec.Command("tmux", "unbind-key", "-n", "S-Up")
	_ = unbindUp.Run()
	unbindDown := exec.Command("tmux", "unbind-key", "-n", "S-Down")
	_ = unbindDown.Run()
	unbindUp = exec.Command("tmux", "unbind-key", "-T", "prefix", "S-Up")
	_ = unbindUp.Run()
	unbindDown = exec.Command("tmux", "unbind-key", "-T", "prefix", "S-Down")
	_ = unbindDown.Run()

	// Bind S-Up to resize up
	cmd := exec.Command("tmux", "bind-key", "-r", "-n", "S-Up", "resize-pane", "-U", "1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to set up Shift+Up binding: %v, stderr: %s", err, stderr.String())
		return err
	}

	// Bind S-Down to resize down
	cmd = exec.Command("tmux", "bind-key", "-r", "-n", "S-Down", "resize-pane", "-D", "1")
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		logger.Error("Failed to set up Shift+Down binding: %v, stderr: %s", err, stderr.String())
		return err
	}

	logger.Debug("Set up Shift+Up and Shift+Down bindings for resizing")
	return nil
}

// TmuxSwapPane swaps the specified pane with the one in the given direction
func TmuxSwapPane(paneId, direction string) error {
	cmd := exec.Command("tmux", "swap-pane", direction, "-t", paneId)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to swap pane %s with direction %s: %v, stderr: %s", paneId, direction, err, stderr.String())
		return err
	}

	logger.Debug("Swapped pane %s with direction %s", paneId, direction)
	return nil
}

// TmuxSelectPane selects the specified pane
func TmuxSelectPane(paneId string) error {
	cmd := exec.Command("tmux", "select-pane", "-t", paneId)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to select pane %s: %v, stderr: %s", paneId, err, stderr.String())
		return err
	}

	logger.Debug("Selected pane %s", paneId)
	return nil
}

// TmuxSetupPaneScrollBindings sets up Ctrl+Up and Ctrl+Down for scrolling the current pane
func TmuxSetupPaneScrollBindings() error {
	// Bind M-Up to scroll up
	cmd := exec.Command("tmux", "bind-key", "-n", "M-Up", "copy-mode", ";", "send-keys", "Up")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to set up M-Up scroll binding: %v, stderr: %s", err, stderr.String())
		return err
	}

	// Bind M-Down to scroll down
	cmd = exec.Command("tmux", "bind-key", "-n", "M-Down", "copy-mode", ";", "send-keys", "Down")
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		logger.Error("Failed to set up M-Down scroll binding: %v, stderr: %s", err, stderr.String())
		return err
	}

	logger.Debug("Set up M-Up and M-Down bindings for scrolling")
	return nil
}
