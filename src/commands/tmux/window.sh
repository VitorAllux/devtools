#!/usr/bin/env bash

# Use directories from .env or fallback to defaults
search_dirs=()
if [ -n "${API_DIR:-}" ] && [ -d "$API_DIR" ]; then search_dirs+=("$API_DIR"); fi
if [ -n "${WEB_DIR:-}" ] && [ -d "$WEB_DIR" ]; then search_dirs+=("$WEB_DIR"); fi

# Default fallback if .env dirs are missing
if [ ${#search_dirs[@]} -eq 0 ]; then
    search_dirs=("$HOME/workspace")
fi

if [[ $# -eq 1 ]]; then
    selected=$1
else
    dirs=$(find "${search_dirs[@]}" \
        -mindepth 1 -maxdepth 3 \
        \( -name ".*" -o -name "node_modules" -o -name "vendor" -o -name "dist" -o -name "public" \) -prune \
        -o -type d -print 2>/dev/null)
        
    options="[.] CURRENT DIR"$'\n'"$dirs"
    
    selected=$(echo "$options" | fzf --bind 'ctrl-y:accept' --prompt="New Window > ")
fi

if [[ -z "$selected" ]]; then
    echo "No dir selected."
    exit 0
fi

if [[ "$selected" == "[.] CURRENT DIR" ]]; then
    selected="$PWD"
fi

if [[ -z $selected ]]; then
    echo "No dir selected."
    exit 0
fi

if [[ ! -d "$selected" ]]; then
    echo "Error: directory not found!"
    exit 1
fi

branch_name=$(basename "$selected" | tr . _)
clean_name=$(echo "$branch_name" | tr "./" "__")

if [ -z "$TMUX" ]; then
    echo "You must be inside a tmux session to run this script."
    exit 1
fi

# Adds a new window to the current session pointing to the selected directory
tmux new-window -n "$clean_name" -c "$selected"
