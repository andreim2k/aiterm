# AI Shell Mode (aish)

## Overview

AI Shell Mode (aish) is a new interactive mode for AITerm that transforms your terminal into an intelligent command-line interface. Instead of switching between a chat pane and execution pane, you work directly in a shell where natural language queries are instantly translated into executable commands.

## Key Features

- **Natural Language to Command Translation**: Type what you want to do in plain English, press `Ctrl+I`, and watch it transform into the appropriate shell command
- **Inline Replacement**: The natural language text is replaced directly in your command line with the generated command
- **Braille Spinner Animation**: A smooth spinner appears at the end of your text while AI processes your request
- **Multiple Options**: When AI finds multiple valid approaches, you can select from a numbered list
- **Direct Execution**: Once the command appears, simply press Enter to execute or Esc to cancel
- **Shell-Aware**: Detects your shell (bash, zsh, fish, etc.) and generates appropriate commands

## Starting AI Shell Mode

```bash
./aiterm --shell
# or
./aiterm -s
```

## How It Works

1. **Type Natural Language**: Write what you want to do in plain English
   ```
   $ list all docker containers that are running
   ```

2. **Press Ctrl+I**: This triggers the AI translation
   ```
   $ list all docker containers that are running ⠋
   ```

3. **View the Command**: The natural language is replaced with the actual command
   ```
   $ docker ps
   ```

4. **Execute or Edit**: Press Enter to run, or edit the command first

## Examples

### Example 1: File Operations
```
Input:  find all python files modified today
Output: find . -name "*.py" -mtime 0
```

### Example 2: System Information
```
Input:  show me disk usage in human readable format
Output: df -h
```

### Example 3: Git Operations
```
Input:  show me the last 5 commits with stats
Output: git log -5 --stat
```

### Example 4: Multiple Options
```
Input:  show disk usage
Output (Multiple options):
  1) df -h
  2) du -sh *
  3) ncdu
  0) Cancel
Select option (0-3):
```

## Workflow

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  Type Natural Language                                      │
│  ↓                                                          │
│  Press Ctrl+I                                               │
│  ↓                                                          │
│  AI Translates (spinner shows progress)                     │
│  ↓                                                          │
│  Command Appears (replaces natural language)                │
│  ↓                                                          │
│  Press Enter to Execute or Edit as needed                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Key Bindings

- **Ctrl+I**: Translate natural language to command
- **Ctrl+C**: Cancel current operation
- **Ctrl+D**: Exit AI Shell mode
- **Enter**: Execute command
- **Up/Down**: Navigate command history (standard shell behavior)

## Configuration

AI Shell mode uses the same configuration as the regular AITerm mode:

```yaml
# ~/.config/aiterm/config.yaml

# Default model to use
default_model: "gpt4"

# Model configurations
models:
  gpt4:
    provider: "openai"
    model: "gpt-4"
    api_key: "${OPENAI_API_KEY}"

  gemini:
    provider: "openrouter"
    model: "google/gemini-2.5-flash-preview"
    api_key: "${OPENROUTER_API_KEY}"
```

## Command Translation Rules

The AI follows these principles when translating natural language to commands:

1. **Safety First**: Generated commands are safe and follow best practices
2. **OS Aware**: Considers your operating system when generating commands
3. **Shell Specific**: Respects shell-specific syntax and features
4. **Concise**: Outputs only the command, no explanations or comments
5. **Multiple Options**: When ambiguous, provides 2-3 alternatives

## Use Cases

### Quick Command Lookup
Instead of googling "how to find large files", just type it and press Ctrl+I:
```
find large files over 100MB → find . -type f -size +100M
```

### Complex Operations
```
compress all log files from last month → find . -name "*.log" -mtime +30 -exec gzip {} \;
```

### System Administration
```
show me top 10 memory consuming processes → ps aux --sort=-%mem | head -n 11
```

### Git Workflows
```
undo last commit but keep changes → git reset --soft HEAD~1
```

## Differences from Regular AITerm Mode

| Feature | Regular Mode | AI Shell Mode |
|---------|--------------|---------------|
| Interface | Split panes (chat + exec) | Single shell pane |
| Interaction | Conversational | Direct translation |
| Context | Full chat history | Single command focus |
| Execution | Via /exec or AI decision | Direct user control |
| Use Case | Complex tasks, assistance | Quick command lookups |

## Tips

1. **Be Specific**: More specific requests yield better commands
   - Good: "list all PDF files in current directory"
   - Okay: "list PDFs"

2. **Use Context**: Include relevant details
   - Good: "find Python files modified in last 24 hours"
   - Okay: "find Python files"

3. **Review Before Execute**: Always check the generated command before pressing Enter

4. **Learn from AI**: AI Shell mode is also a learning tool - see how natural requests map to commands

## Troubleshooting

### Command Not What You Expected
- Press Ctrl+C to cancel
- Retype with more specific details
- Or edit the command manually before executing

### Multiple Options Every Time
- The AI detects ambiguity in your request
- Choose the option that best fits your need
- For future requests, be more specific

### Slow Response
- Check your AI API configuration
- Verify your internet connection
- Consider using a faster model (like Gemini Flash)

## Exit AI Shell Mode

Type `exit` or press `Ctrl+D` to quit AI Shell mode and return to your regular shell.

## Comparison with Chat Mode

**Use AI Shell Mode when:**
- You need quick command translations
- You want direct control over execution
- You know what you want to do but not the exact command
- You're looking up syntax or command options

**Use Regular Chat Mode when:**
- You need troubleshooting help
- You want AI to analyze output
- You need multi-step operations
- You want conversational assistance

## Future Enhancements

Potential improvements planned:
- Command history persistence
- Command explanation mode (show what the command does)
- Command validation warnings
- Integration with shell history
- Alias suggestions based on frequent patterns
