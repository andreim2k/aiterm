package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/andreim2k/aiterm/config"
	"github.com/andreim2k/aiterm/logger"
	"github.com/andreim2k/aiterm/system"
	"github.com/fatih/color"
)

type AIResponse struct {
	Message                string
	SendKeys               []string
	ExecCommand            []string
	PasteMultilineContent  string
	RequestAccomplished    bool
	ExecPaneSeemsBusy      bool
	WaitingForUserResponse bool
	NoComment              bool
}

// Parsed only when pane is prepared
type CommandExecHistory struct {
	Command string
	Output  string
	Code    int
}

// Manager represents the AITerm manager agent
type Manager struct {
	Config           *config.Config
	AiClient         *AiClient
	Status           string // running, waiting, done
	PaneId           string
	ExecPane         *system.TmuxPaneDetails
	Messages         []ChatMessage
	ExecHistory      []CommandExecHistory
	WatchMode        bool
	ShellMode        bool // AI Shell mode (aish)
	OS               string
	SessionOverrides map[string]interface{} // session-only config overrides
	LoadedKBs        map[string]string      // Loaded knowledge bases (name -> content)

	// Functions for mocking
	confirmedToExec   func(command string, prompt string, edit bool) (bool, string)
	getTmuxPanesInXml func(config *config.Config) string
}

// NewManager creates a new manager agent
func NewManager(cfg *config.Config, shellMode bool) (*Manager, error) {

	paneId, err := system.TmuxCurrentPaneId()
	if err != nil {
		// If we're not in a tmux session, exec into tmux directly to avoid background job
		args := strings.Join(os.Args[1:], " ")
		binaryName := os.Args[0]

		// Use syscall.Exec to replace current process with tmux
		// This avoids creating a background job in the parent shell
		tmuxCmd := []string{
			"tmux", "new-session",
			fmt.Sprintf("%s %s", binaryName, args),
		}

		err = system.TmuxExecSession(tmuxCmd)
		if err != nil {
			return nil, fmt.Errorf("system.TmuxExecSession failed: %w", err)
		}
		// This line should never be reached because exec replaces the process
		os.Exit(0)
	}

	aiClient := NewAiClient(cfg)
	os := system.GetOSDetails()

	manager := &Manager{
		Config:           cfg,
		AiClient:         aiClient,
		PaneId:           paneId,
		Messages:         []ChatMessage{},
		ExecPane:         &system.TmuxPaneDetails{},
		ShellMode:        shellMode,
		OS:               os,
		SessionOverrides: make(map[string]interface{}),
		LoadedKBs:        make(map[string]string),
	}

	// Set the config manager in the AI client
	aiClient.SetConfigManager(manager)

	manager.confirmedToExec = manager.confirmedToExecFn
	manager.getTmuxPanesInXml = manager.getTmuxPanesInXmlFn

	// Set up tmux styling
	_ = system.TmuxSetupStyling()

	// In shell mode, we don't create an exec pane
	if !shellMode {
		manager.InitExecPane()
		// Set chat pane title
		_ = system.TmuxSetPaneTitle(paneId, " ai chat ")
	} else {
		// In shell mode, set a different pane title
		_ = system.TmuxSetPaneTitle(paneId, " ai shell ")
	}

	// Update status bar with current model
	manager.updateStatusBar()

	// Auto-load knowledge bases from config
	manager.autoLoadKBs()

	return manager, nil
}

// NewManagerForTranslation creates a minimal manager for AI command translation
// This bypasses all tmux initialization for use in shell mode translation
func NewManagerForTranslation(cfg *config.Config) (*Manager, error) {
	aiClient := NewAiClient(cfg)
	os := system.GetOSDetails()

	manager := &Manager{
		Config:           cfg,
		AiClient:         aiClient,
		PaneId:           "", // No pane ID needed for translation
		Messages:         []ChatMessage{},
		ExecPane:         &system.TmuxPaneDetails{},
		ShellMode:        true, // Set to true to indicate this is for shell mode
		OS:               os,
		SessionOverrides: make(map[string]interface{}),
		LoadedKBs:        make(map[string]string),
	}

	// Set the config manager in the AI client
	aiClient.SetConfigManager(manager)

	manager.confirmedToExec = manager.confirmedToExecFn
	manager.getTmuxPanesInXml = manager.getTmuxPanesInXmlFn

	return manager, nil
}

// Start starts the manager agent
func (m *Manager) Start(initMessage string) error {
	if m.ShellMode {
		shellInterface := NewShellInterface(m)
		if err := shellInterface.Start(); err != nil {
			logger.Error("Failed to start Shell interface: %v", err)
			return err
		}
	} else {
		cliInterface := NewCLIInterface(m)
		if initMessage != "" {
			logger.Info("Initial task provided: %s", initMessage)
		}
		if err := cliInterface.Start(initMessage); err != nil {
			logger.Error("Failed to start CLI interface: %v", err)
			return err
		}
	}

	return nil
}

func (m *Manager) Println(msg string) {
	fmt.Println(m.GetPrompt() + msg)
}

func (m *Manager) GetConfig() *config.Config {
	return m.Config
}

// getPrompt returns the prompt string with color
func (m *Manager) GetPrompt() string {
	arrowColor := color.New(color.FgYellow, color.Bold)
	stateColor := color.New(color.FgMagenta, color.Bold)
	modelColor := color.New(color.FgCyan, color.Bold)
	doneColor := color.New(color.FgCyan, color.Bold)

	var stateSymbol string
	switch m.Status {
	case "running":
		stateSymbol = "▶"
	case "waiting":
		stateSymbol = "?"
	case "done":
		stateSymbol = "✓"
	default:
		stateSymbol = ""
	}
	if m.WatchMode {
		stateSymbol = "∞"
	}

	prompt := ""

	// Show current model if it's not the default or first available model
	currentModel := m.GetModelsDefault()
	availableModels := m.GetAvailableModels()
	if len(availableModels) > 0 {
		// Get the "expected" model (configured default or first available)
		expectedModel := m.Config.DefaultModel
		if expectedModel == "" && len(availableModels) > 0 {
			expectedModel = availableModels[0] // First model as default
		}

		// Show model if current is different from expected
		if currentModel != "" && currentModel != expectedModel {
			prompt += modelColor.Sprint("["+currentModel+"]") + " "
		}
	}

	if stateSymbol == "✓" {
		prompt += doneColor.Sprint("["+stateSymbol+"]") + " "
	} else if stateSymbol != "" {
		prompt += stateColor.Sprint("["+stateSymbol+"]") + " "
	}
	prompt += arrowColor.Sprint("» ")
	return prompt
}

// SwitchPane switches between the chat pane and exec pane
func (m *Manager) SwitchPane() error {
	return system.TmuxSwitchToOtherPane(m.PaneId, m.ExecPane.Id)
}

// updateStatusBar updates the tmux status bar with current model information
func (m *Manager) updateStatusBar() {
	modelName := m.GetModelsDefault()
	provider := ""

	if modelConfig, exists := m.GetCurrentModelConfig(); exists {
		provider = modelConfig.Provider
	}

	if modelName == "" {
		modelName = "default"
	}
	if provider == "" {
		provider = "openrouter"
	}

	_ = system.TmuxUpdateStatusBar(modelName, provider)
}

// TogglePaneCollapse toggles between collapsed and expanded chat pane state
func TogglePaneCollapse(chatPaneId, execPaneId string) error {
	return system.TmuxTogglePaneCollapse(chatPaneId, execPaneId)
}

// CleanupPanes kills the exec pane when aiterm exits
func (m *Manager) CleanupPanes() {
	if m.ExecPane != nil && m.ExecPane.Id != "" {
		_ = system.TmuxKillPane(m.ExecPane.Id)
	}
}

func (ai *AIResponse) String() string {
	return fmt.Sprintf(`
	Message: %s
	SendKeys: %v
	ExecCommand: %v
	PasteMultilineContent: %s
	RequestAccomplished: %v
	ExecPaneSeemsBusy: %v
	WaitingForUserResponse: %v
	NoComment: %v
`,
		ai.Message,
		ai.SendKeys,
		ai.ExecCommand,
		ai.PasteMultilineContent,
		ai.RequestAccomplished,
		ai.ExecPaneSeemsBusy,
		ai.WaitingForUserResponse,
		ai.NoComment,
	)
}
