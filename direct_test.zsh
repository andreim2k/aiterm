#!/bin/zsh

# Simple function
test-widget() {
    echo "Widget called!" >&2
    BUFFER="echo 'Widget worked!'"
    CURSOR=${#BUFFER}
    zle reset-prompt
}

# Register widget
zle -N test-widget

# Bind to Ctrl+Space
bindkey '^@' test-widget

# Bind to Ctrl+X Ctrl+A
bindkey '^X^A' test-widget

echo "Test widget loaded. Try Ctrl+Space or Ctrl+X Ctrl+A"
