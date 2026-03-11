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
    # 1. Grab directories
    # 2. Prepend a back/cancel option to the list
    dirs=$(find "${search_dirs[@]}" \
        -mindepth 1 -maxdepth 3 \
        \( -name ".*" -o -name "node_modules" -o -name "vendor" -o -name "dist" -o -name "public" \) -prune \
        -o -type d -print 2>/dev/null)
        
    options="[.] CURRENT DIR"$'\n'"$dirs"
    
    selected=$(echo "$options" | fzf --bind 'ctrl-y:accept' --prompt="Start Session > ")
fi

if [[ -z "$selected" ]]; then
    exit 0
fi

if [[ "$selected" == "[.] CURRENT DIR" ]]; then
    selected="$PWD"
fi

if [[ -z $selected ]]; then
    exit 0
fi

base_name=$(basename "$selected" | tr . _ | tr -c '[:alnum:]' '_')
base_name="${base_name%%_}"
window_name=$base_name

session_name="space"
index=1

while tmux has-session -t="$session_name" 2>/dev/null; do
    session_name="space_${index}"
    ((index++))
done

tmux new-session -ds "$session_name" -n "$window_name" -c "$selected"

if [ -n "$TMUX" ]; then
    tmux switch-client -t "$session_name"
else
    tmux attach -t "$session_name"
fi
