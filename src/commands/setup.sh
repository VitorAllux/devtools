#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"

title "Setting up devtools Environment"

info "Installing dependencies via apt (requires sudo)..."
sudo apt-get update -y
sudo apt-get install -y zsh htop curl wget git jq pv rclone mysql-client gzip fzf xclip wl-clipboard python3-pip pipx

if [[ "$SHELL" != "/usr/bin/zsh" ]]; then
  info "Setting ZSH as the default shell..."
  sudo chsh -s "$(which zsh)" "$USER" || true
else
  ok "ZSH is already the default shell."
fi

info "Configuring Native ZSH Autocompletion for devv..."
mkdir -p ~/.zfunc
ln -sf "${DEVTOOLS_DIR}/completions/_devv" ~/.zfunc/_devv

if ! grep -q "fpath+=(~/.zfunc)" ~/.zshrc 2>/dev/null; then
  echo '' >> ~/.zshrc
  echo '# devtools autocomplete' >> ~/.zshrc
  echo 'fpath+=(~/.zfunc)' >> ~/.zshrc
  echo 'autoload -Uz compinit && compinit' >> ~/.zshrc
fi

# Clean up any lingering old auto-completion aliases if they somehow managed to survive
sed -i '/compdef devv=just/d' ~/.zshrc 2>/dev/null || true
sed -i '/_devv() { words=/d' ~/.zshrc 2>/dev/null || true
sed -i '/compdef _devv devv/d' ~/.zshrc 2>/dev/null || true

info "Configuring devv Keybindings (Tmux FZF)..."
if ! grep -q "devv tmux:session" ~/.zshrc 2>/dev/null; then
  echo 'bindkey -s ^f "devv tmux:session\n"' >> ~/.zshrc
fi
if ! grep -q "devv tmux:window" ~/.zshrc 2>/dev/null; then
  echo 'bindkey -s ^w "devv tmux:window\n"' >> ~/.zshrc
fi
if ! grep -q "devv db:ui" ~/.zshrc 2>/dev/null; then
  echo 'bindkey -s ^g "devv db:ui\n"' >> ~/.zshrc
fi

info "Ensuring global PATH access in .zshrc and .bashrc..."
BIN_DIR="${HOME}/workspace/bin"
mkdir -p "$BIN_DIR"

if ! grep -q "workspace/bin" ~/.zshrc 2>/dev/null; then
  echo 'export PATH="$HOME/workspace/bin:$PATH"' >> ~/.zshrc
fi
if ! grep -q "workspace/bin" ~/.bashrc 2>/dev/null; then
  echo 'export PATH="$HOME/workspace/bin:$PATH"' >> ~/.bashrc
fi

# Create symlink instead of copying so it dynamically updates
ln -sf "${DEVTOOLS_DIR}/bin/devv" "${BIN_DIR}/devv"
chmod +x "${DEVTOOLS_DIR}/bin/devv"
find "${DEVTOOLS_DIR}/src/commands" -name "*.sh" -exec chmod +x {} +

ok "Setup complete!"
info "Please restart your terminal or run 'exec zsh' to apply changes."
