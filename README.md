<br/>
<div align="center">
  <a href="https://github.com/andreim2k/aiterm">
    <img src="https://aiterm.dev/gh.svg?v=2" alt="AITerm Logo" width="100%">
  </a>
  <h3 align="center">AITerm</h3>
  <p align="center">
    Your intelligent pair programmer directly within your tmux sessions.
    <br/>
    <br/>
    <a href="https://github.com/andreim2k/aiterm/blob/main/LICENSE"><img alt="License" src="https://img.shields.io/github/license/andreim2k/aiterm?style=flat-square"></a>
    <a href="https://github.com/andreim2k/aiterm/releases/latest"><img alt="Release" src="https://img.shields.io/github/v/release/andreim2k/aiterm?style=flat-square"></a>
    <a href="https://github.com/andreim2k/aiterm/issues"><img alt="Issues" src="https://img.shields.io/github/issues/andreim2k/aiterm?style=flat-square"></a>
    <br/>
    <br/>
    <br/>
    <a href="https://aiterm.dev/screenshots" target="_blank">Screenshots</a> |
    <a href="https://github.com/andreim2k/aiterm/issues/new?labels=bug&template=bug_report.md" target="_blank">Report Bug</a> |
    <a href="https://github.com/andreim2k/aiterm/issues/new?labels=enhancement&template=feature_request.md" target="_blank">Request Feature</a>
    <br/>
    <br/>
    <a href="https://aiterm.dev/tmux-cheat-sheet/" target="_blank">Tmux Cheat Sheet</a> |
    <a href="https://aiterm.dev/tmux-getting-started/" target="_blank">Tmux Getting Started</a> |
    <a href="https://aiterm.dev/tmux-config/" target="_blank">Tmux Config Generator</a>
  </p>
</div>

## Table of Contents

- [About The Project](#about-the-project)
  - [Human-Inspired Interface](#human-inspired-interface)
- [Installation](#installation)
  - [Quick Install](#quick-install)
  - [Homebrew](#homebrew)
  - [Manual Download](#manual-download)
  - [Install from Main](#install-from-main)
- [Post-Installation Setup](#post-installation-setup)
- [AITerm Layout](#aiterm-layout)
- [Observe Mode](#observe-mode)
- [Prepare Mode](#prepare-mode)
- [Watch Mode](#watch-mode)
  - [Activating Watch Mode](#activating-watch-mode)
  - [Example Use Cases](#example-use-cases)
- [Knowledge Base](#knowledge-base)
  - [Creating Knowledge Bases](#creating-knowledge-bases)
  - [Using Knowledge Bases](#using-knowledge-bases)
  - [Auto-Loading Knowledge Bases](#auto-loading-knowledge-bases)
- [Model Configuration](#model-configuration)
  - [Setting Up Multiple Models](#setting-up-multiple-models)
  - [Switching Between Models](#switching-between-models)
- [Squashing](#squashing)
  - [What is Squashing?](#what-is-squashing)
  - [Manual Squashing](#manual-squashing)
- [Keybindings](#keybindings)
- [Core Commands](#core-commands)
- [Command-Line Usage](#command-line-usage)
- [Configuration](#configuration)
  - [Environment Variables](#environment-variables)
  - [Session-Specific Configuration](#session-specific-configuration)
- [Contributing](#contributing)
- [License](#license)

## About The Project

![Product Demo](https://aiterm.dev/demo.png)

AITerm is an intelligent terminal assistant that lives inside your tmux sessions. Unlike other CLI AI tools, AITerm observes and understands the content of your tmux panes, providing assistance without requiring you to change your workflow or interrupt your terminal sessions.

Think of AITerm as a _pair programmer_ that sits beside you, watching your terminal environment exactly as you see it. It can understand what you're working on across multiple panes, help solve problems and execute commands on your behalf in a dedicated execution pane.

### Human-Inspired Interface

AITerm's design philosophy mirrors the way humans collaborate at the terminal. Just as a colleague sitting next to you would observe your screen, understand context from what's visible, and help accordingly, AITerm:

1. **Observes**: Reads the visible content in all your panes
2. **Communicates**: Uses a dedicated chat pane for interaction
3. **Acts**: Can execute commands in a separate execution pane (with your permission)

This approach provides powerful AI assistance while respecting your existing workflow and maintaining the familiar terminal environment you're already comfortable with.

## Installation

AITerm requires only tmux to be installed on your system. It's designed to work on Unix-based operating systems including Linux and macOS.

### Quick Install

The fastest way to install AITerm is using the installation script:

```bash
# install tmux if not already installed
curl -fsSL https://get.aiterm.dev | bash
```

This installs AITerm to `/usr/local/bin/aiterm` by default. If you need to install to a different location or want to see what the script does before running it, you can view the source at [get.aiterm.dev](https://get.aiterm.dev).

### Manual Download

You can also download pre-built binaries from the [GitHub releases page](https://github.com/andreim2k/aiterm/releases).

After downloading, make the binary executable and move it to a directory in your PATH:

```bash
chmod +x ./aiterm
sudo mv ./aiterm /usr/local/bin/
```

### Install from Main

To install the latest development version directly from the main branch:

```bash
go install github.com/andreim2k/aiterm@main
```

**Note:** The main branch contains the latest features and fixes but may be less stable than official releases.

## Post-Installation Setup

AITerm reads its configuration from `~/.config/aiterm/config.yaml`. To get running, create the file with a model entry that points at the provider you use.

1. **Create the config path**

   ```bash
   mkdir -p ~/.config/aiterm
   vim ~/.config/aiterm/config.yaml
   ```

2. **Add a minimal config**

   ```yaml
   models:
     primary:
       provider: openrouter  # openrouter, openai or azure
       model: anthropic/claude-haiku-4.5
       api_key: sk-your-api-key
   ```

   Swap the provider name and fill in the model/API key required by your account.

3. **Start AITerm**

   ```bash
   aiterm
   ```

See [Model Configuration](#model-configuration) for more details.

## AITerm Layout

![Panes](https://aiterm.dev/shots/panes.png?lastmode=1)

AITerm is designed to operate within a single tmux window, with one instance of
AITerm running per window and organizes your workspace using the following pane structure:

1. **Chat Pane**: This is where you interact with the AI. It features a REPL-like interface with syntax highlighting, auto-completion, and readline shortcuts.

2. **Exec Pane**: AITerm selects (or creates) a pane where commands can be executed.

3. **Read-Only Panes**: All other panes in the current window serve as additional context. AITerm can read their content but does not interact with them.

## Observe Mode

![Observe Mode](https://aiterm.dev/shots/demo-observe.png)
_AITerm sent the first ping command and is waiting for the countdown to check for the next step_

AITerm operates by default in "observe mode". Here's how the interaction flow works:

1. **User types a message** in the Chat Pane.

2. **AITerm captures context** from all visible panes in your current tmux window (excluding the Chat Pane itself). This includes:

   - Current command with arguments
   - Detected shell type
   - User's operating system
   - Current content of each pane

3. **AITerm processes your request** by sending user's message, the current pane context, and chat history to the AI. A spinning animation indicator appears during processing.

4. **The AI responds** with information, which may include a suggested command to run.

5. **If a command is suggested**, AITerm will:

   - Check if the command matches whitelist or blacklist patterns
   - Ask for your confirmation (unless the command is whitelisted). The confirmation prompt includes a risk indicator (✓ safe, ? unknown, ! danger) for guidance only - always review commands carefully as the risk scoring is not exhaustive and should not be relied upon for security decisions
   - Execute the command in the designated Exec Pane if approved
   - Wait for the `wait_interval` (default: 5 seconds) (You can pause/resume the countdown with `space` or `enter` to stop the countdown)
   - Capture the new output from all panes
   - Send the updated context back to the AI to continue helping you

6. **The conversation continues** until your task is complete.

![Observe Mode Flowchart](https://aiterm.dev/shots/observe-mode.png)

## Prepare Mode

![Prepare Mode](https://aiterm.dev/shots/demo-prepare.png?lastmode=1)
_AITerm customized the pane prompt and sent the first ping command. Instead of the countdown, it's waiting for command completion_

Prepare mode is an optional feature that enhances AITerm's ability to work with your terminal by customizing
your shell prompt and tracking command execution with better precision. This
enhancement eliminates the need for arbitrary wait intervals and provides the AI
with more detailed information about your commands and their results.

When you enable Prepare Mode, AITerm will:

1. **Detects your current shell** in the execution pane (supports bash, zsh, and fish)
2. **Customizes your shell prompt** to include special markers that AITerm can recognize
3. **Will track command execution history** including exit codes, and per-command outputs
4. **Will detect command completion** instead of using fixed wait time intervals

To activate Prepare Mode, simply use:

```
AITerm » /prepare
```

By default, AITerm will attempt to detect the shell running in the execution pane. If you need to specify the shell manually, you can provide it as an argument:

```
AITerm » /prepare bash
```

**Prepared Fish Example:**

```shell
$ function fish_prompt; set -l s $status; printf '%s@%s:%s[%s][%d]» ' $USER (hostname -s) (prompt_pwd) (date +"%H:%M") $s; end
username@hostname:~/r/aiterm[21:05][0]»
```

## Watch Mode

![Watch Mode](https://aiterm.dev/shots/demo-watch.png)
_AITerm watching user shell commands and better alternatives_

Watch Mode transforms AITerm into a proactive assistant that continuously
monitors your terminal activity and provides suggestions based on what you're
doing.

### Activating Watch Mode

To enable Watch Mode, use the `/watch` command followed by a description of what you want AITerm to look for:

```
AITerm » /watch spot and suggest more efficient alternatives to my shell commands
```

When activated, AITerm will:

1. Start capturing the content of all panes in your current tmux window at regular intervals (`wait_interval` configuration)
2. Analyze content based on your specified watch goal and provide suggestions when appropriate

### Example Use Cases

Watch Mode could be valuable for scenarios such as:

- **Learning shell efficiency**: Get suggestions for more concise commands as you work

  ```
  AITerm » /watch spot and suggest more efficient alternatives to my shell commands
  ```

- **Detecting common errors**: Receive warnings about potential issues or mistakes

  ```
  AITerm » /watch flag commands that could expose sensitive data or weaken system security
  ```

- **Log Monitoring and Error Detection**: Have AITerm monitor log files or terminal output for errors

  ```
  AITerm » /watch monitor log output for errors, warnings, or critical issues and suggest fixes
  ```

## Squashing

As you work with AITerm, your conversation history grows, adding to the context
provided to the AI model with each interaction. Different AI models have
different context size limits and pricing structures based on token usage. To
manage this, AITerm implements a simple context management feature called
"squashing."

### What is Squashing?

Squashing is AITerm's built-in mechanism for summarizing chat history to manage
token usage.

When your context grows too large, AITerm condenses previous
messages into a more compact summary.

You can check your current context utilization at any time using the `/info` command:

```bash
AITerm » /info

Context
────────

Messages            15
Context Size~       82500 tokens
                    ████████░░ 82.5%
Max Size            100000 tokens
```

This example shows that the context is at 82.5% capacity (82,500 tokens out of 100,000). When the context size reaches 80% of the configured maximum (`max_context_size` in your config), AITerm automatically triggers squashing.

### Manual Squashing

If you'd like to manage your context before reaching the automatic threshold, you can trigger squashing manually with the `/squash` command:

```bash
AITerm » /squash
```

## Knowledge Base

The Knowledge Base feature allows you to create pre-defined context files in markdown format that can be loaded into AITerm's conversation context. This is useful for sharing common patterns, workflows, or project-specific information with the AI across sessions.

### Creating Knowledge Bases

Knowledge bases are text files stored in `~/.config/aiterm/kb/`. To create one:

1. Create the knowledge base directory if it doesn't exist:
   ```bash
   mkdir -p ~/.config/aiterm/kb
   ```

2. Create a file with your knowledge base content:
   ```bash
   cat > ~/.config/aiterm/kb/docker-workflows << 'EOF'
   # Docker Workflows

   ## Common Commands
   - Always use `docker compose` (not `docker-compose`)
   - Prefer named volumes over bind mounts for databases
   - Use `.env` files for environment-specific configuration

   ## Project Structure
   - Development: `docker compose -f docker-compose.dev.yml up`
   - Production: `docker compose -f docker-compose.prod.yml up -d`
   EOF
   ```

### Using Knowledge Bases

Once created, you can load knowledge bases into your AITerm session:

```bash
# List available knowledge bases
AITerm » /kb
Available knowledge bases:
  [ ] docker-workflows
  [ ] git-conventions
  [ ] testing-procedures

# Load a knowledge base
AITerm » /kb load docker-workflows
✓ Loaded knowledge base: docker-workflows (850 tokens)

# List again to see loaded status
AITerm » /kb
Available knowledge bases:
  [✓] docker-workflows (850 tokens)
  [ ] git-conventions
  [ ] testing-procedures

Loaded: 1 KB(s), 850 tokens

# Unload a knowledge base
AITerm » /kb unload docker-workflows
✓ Unloaded knowledge base: docker-workflows

# Unload all knowledge bases
AITerm » /kb unload --all
✓ Unloaded all knowledge bases (2 KB(s))
```

You can also load knowledge bases directly from the command line when starting AITerm:

```bash
# Load single knowledge base
aiterm --kb docker-workflows

# Load multiple knowledge bases (comma-separated)
aiterm --kb docker-workflows,git-conventions
```

### Auto-Loading Knowledge Bases

You can configure knowledge bases to load automatically on startup by adding them to your `~/.config/aiterm/config.yaml`:

```yaml
knowledge_base:
  auto_load:
    - docker-workflows
    - git-conventions
  # path: /custom/path  # Optional: use custom KB directory
```

**Important Notes:**
- Loaded knowledge bases consume tokens from your context budget
- Use `/info` to see how many tokens your loaded KBs are using
- Knowledge bases are injected after the system prompt but before conversation history
- Unloading a KB removes it from future messages immediately

## Model Configuration

AITerm supports configuring multiple AI model configurations and easily switching between them. This allows you to define different AI providers, models, and settings for various use cases.

### Setting Up Multiple Models

Configure multiple AI models in your `~/.config/aiterm/config.yaml`:

```yaml
# Optional: specify which model to use by default
# If not set, the first model alphabetically will be used automatically
default_model: "fast"

models:
  fast:
    provider: "openrouter"
    model: "anthropic/claude-haiku-4.5"
    api_key: "sk-or-your-openrouter-key"

  smart:
    provider: "openrouter"
    model: "google/gemini-2.5-prod"
    api_key: "sk-or-your-openrouter-key"

  # You can use any chat completion compatible endpoint as base_url
  anthropic:
    provider: "openrouter"
    model: "claude-3-5-sonnet-20241022"
    api_key: "your-anthropic-api-key"
    base_url: "https://api.anthropic.com"

  github-copilot:
    provider: "openrouter"
    model: "claude-sonnet-4.5"
    api_key: "your-github-copilot-api-key"
    base_url: "https://api.githubcopilot.com"

  local-llama:
    provider: "openrouter"
    model: "gemma3:1b"
    api_key: "sk-or-your-openrouter-key"
    base_url: http://localhost:11434/v1

  # Requesty.ai (no base_url needed - uses default)
  requesty-smart:
    provider: "requesty"
    model: "smart/task"
    api_key: "your-requesty-api-key"

  # xAI Grok (no base_url needed - uses default)
  grok:
    provider: "xai"
    model: "grok-beta"
    api_key: "your-xai-api-key"

  # Alibaba Cloud (no base_url needed - uses default)
  qwen:
    provider: "alibaba"
    model: "qwen-max"
    api_key: "your-alibaba-api-key"

  # Responses API
  codex:
    provider: "openai"
    model: "gpt-5-codex"
    api_key: "sk-or-your-openrouter-key"

  azure-gpt4:
    provider: "azure"
    model: "gpt-4o"
    api_key: "your-azure-openai-api-key"
    api_base: "https://your-resource.openai.azure.com/"
    api_version: "2025-04-01-preview"
    deployment_name: "gpt-4o"
```

**Supported Providers:**
- `openai` - OpenAI Responses API (GPT-4, GPT-5, etc.)
- `openrouter` - Universal Chat Completion API (default base URL: https://openrouter.ai/api/v1)
- `requesty` - Requesty.ai Router (default base URL: https://router.requesty.ai/v1)
- `zai` - zAI API (default base URL: https://api.zai.com/v1)
- `xai` - xAI/Grok API (default base URL: https://api.x.ai/v1)
- `alibaba` - Alibaba Cloud DashScope (default base URL: https://dashscope.aliyuncs.com/compatible-mode/v1)
- `azure` - Azure Chat Completions API

**Note:** For providers with default base URLs (requesty, zai, xai, alibaba, openrouter), you don't need to specify `base_url` in your configuration unless you want to override it.

**Interactive Commands:**
```bash
# List available models and see current selection
AITerm » /model

Available Models
  [ ] claude-sonnet (openrouter: anthropic/claude-3.5-sonnet)
  [ ] fast (openrouter: anthropic/claude-haiku-4.5)
  [✓] smart (openrouter: google/gemini-2.5-prod)
  [ ] local-llama (openrouter: meta-llama/llama-3.1-8b-instruct:free)

Current Model:
  Configuration: smart
  Provider: openrouter
  Model: google/gemini-2.5-prod

# Switch to a different model
AITerm » /model claude-sonnet
✓ Switched to claude-sonnet (openrouter: anthropic/claude-3.5-sonnet)

# Status bar shows current model when using non-default
AITerm [claude-sonnet] »
```

## Keybindings

AITerm provides convenient keyboard shortcuts to enhance your workflow:

| Keybinding | Description |
| ---------- | ----------- |
| `Shift+Tab` | Switch focus between the AITerm chat pane and execution pane |
| `Shift+Down` | Toggle chat pane collapse/expand - press to hide the chat pane and work in full shell, press again to bring it back |
| `Tab` | Auto-complete commands and options in the chat pane |

All keybindings are configured at the tmux level, allowing you to quickly control panes without interrupting your workflow.

**Workflow Tips:**
- Use `Shift+Down` to collapse the chat pane when you need full terminal space for your shell work
- Press `Shift+Down` again to expand the chat pane and continue your AI conversation
- Use `Shift+Tab` to quickly switch focus between panes without changing their size

## Core Commands

| Command                     | Description                                                      |
| --------------------------- | ---------------------------------------------------------------- |
| `/info`                     | Display system information, pane details, and context statistics |
| `/clear`                    | Clear chat history.                                              |
| `/reset`                    | Clear chat history and reset all panes.                          |
| `/config`                   | View current configuration settings                              |
| `/config set <key> <value>` | Override configuration for current session                       |
| `/model`                    | List available models and show current active model              |
| `/model <name>`             | Switch to a different model configuration                        |
| `/squash`                   | Manually trigger context summarization                           |
| `/prepare [shell]`          | Initialize Prepared Mode for the Exec Pane (e.g., bash, zsh)    |
| `/watch <description>`      | Enable Watch Mode with specified goal                            |
| `/kb`                       | List available knowledge bases with loaded status                |
| `/kb load <name>`           | Load a knowledge base into conversation context                  |
| `/kb unload <name>`         | Unload a specific knowledge base                                 |
| `/kb unload --all`          | Unload all knowledge bases                                       |
| `/exit`                     | Exit AITerm                                                      |

## Command-Line Usage

You can start `aiterm` with an initial message, task file, model configuration, or knowledge bases from the command line:

- **Direct Message:**

  ```sh
  aiterm your initial message
  ```

- **Task File:**
  ```sh
  aiterm -f path/to/your_task.txt
  ```

- **Specify Model:**
  ```sh
  # Use a specific model configuration
  aiterm --model gpt4 "Write a Go function"
  aiterm --model claude-sonnet
  ```

- **Load Knowledge Bases:**
  ```sh
  # Single knowledge base
  aiterm --kb docker-workflows

  # Multiple knowledge bases
  aiterm --kb docker-workflows,git-conventions
  ```

- **Combine Options:**
  ```sh
  aiterm --model gpt4 --kb docker-workflows "Debug this Docker issue"
  ```

## Configuration

The configuration can be managed through a YAML file, environment variables, or via runtime commands.

AITerm looks for its configuration file at `~/.config/aiterm/config.yaml`.
For a sample configuration file, see [config.example.yaml](https://github.com/andreim2k/aiterm/blob/main/config.example.yaml).

### Environment Variables

All configuration options can also be set via environment variables, which take precedence over the config file. Use the prefix `TMUXAI_` followed by the uppercase configuration key:

```bash
# General settings
export TMUXAI_DEBUG=true
export TMUXAI_MAX_CAPTURE_LINES=300
export TMUXAI_MAX_CONTEXT_SIZE=150000

# Quick setup with environment variables (alternative to model configurations)
export TMUXAI_OPENAI_API_KEY="your-openai-api-key-here"
export TMUXAI_OPENAI_MODEL="gpt-4"
export TMUXAI_OPENROUTER_API_KEY="your-openrouter-api-key-here"
```

You can also use environment variables directly within your configuration file values. The application will automatically expand these variables when loading the configuration:

```yaml
# Example config.yaml with environment variable expansion
openai:
  api_key: "${OPENAI_API_KEY}"
  model: "${OPENAI_MODEL:-gpt-4}"

openrouter:
  api_key: "${OPENROUTER_API_KEY}"
  base_url: "${OPENROUTER_BASE_URL:-https://openrouter.ai/api/v1}"
```

### Session-Specific Configuration

You can override configuration values for your current AITerm session using the `/config` command:

```bash
# View current configuration
AITerm » /config

# Override a configuration value for this session
AITerm » /config set max_capture_lines 300
AITerm » /config set wait_interval 3
```

These changes will persist only for the current session and won't modify your config file.

## Contributing

If you have a suggestion that would make this better, please fork the repo and create a pull request.
You can also simply open an issue.
<br>
Don't forget to give the project a star!

## License

Distributed under the Apache License. See [Apache License](https://github.com/andreim2k/aiterm/blob/main/LICENSE) for more information.
