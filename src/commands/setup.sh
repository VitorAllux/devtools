#!/usr/bin/env bash
source "${DEVTOOLS_DIR}/src/lib/ui.sh"
source "${DEVTOOLS_DIR}/src/lib/secrets.sh"

title "Setting up devtools Environment"

info "Installing dependencies via apt (requires sudo)..."
sudo apt-get update -y
sudo apt-get install -y zsh htop curl wget git jq pv rclone mysql-client gzip fzf xclip wl-clipboard python3-pip python3-tk pipx age

if ! command -v bw >/dev/null 2>&1; then
  warn "Bitwarden CLI (bw) not found. It is optional, but recommended for secret restore during env:bootstrap."

  if [[ -t 0 ]]; then
    if confirm "Do you want to install Bitwarden CLI now? [y/N]"; then
      info "Installing Bitwarden CLI..."

      if ! command -v node >/dev/null 2>&1 || ! command -v npm >/dev/null 2>&1; then
        info "Installing nodejs and npm..."
        sudo apt-get install -y nodejs npm
      fi

      sudo npm install -g @bitwarden/cli

      if command -v bw >/dev/null 2>&1; then
        ok "Bitwarden CLI installed successfully."
      else
        warn "Could not validate Bitwarden CLI installation. Install manually later if needed."
      fi
    else
      info "Skipping Bitwarden CLI install. You can install it later for automatic secret restore."
    fi
  else
    info "Non-interactive mode detected. Skipping optional Bitwarden CLI install."
  fi
fi

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

info "Removing legacy devv shell keybindings to avoid editor/terminal conflicts..."
sed -i '/bindkey -s \^f "devv tmux:session\\n"/d' ~/.zshrc 2>/dev/null || true
sed -i '/bindkey -s \^w "devv tmux:window\\n"/d' ~/.zshrc 2>/dev/null || true
sed -i '/bindkey -s \^g "devv db:ui\\n"/d' ~/.zshrc 2>/dev/null || true
sed -i '/bindkey -s .*\(devv ssh:connect\\n\|devv ssh\\n\)/d' ~/.zshrc 2>/dev/null || true

info "Configuring devv SSH shortcut (Alt+S)..."
echo 'bindkey -s "\es" "devv ssh\n"' >> ~/.zshrc

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
if [[ -f "${DEVV_ENCRYPTED_SERVERS_FILE}" && ! -s "${DEVV_SERVERS_FILE}" ]]; then
  info "Encrypted secrets detected. Run 'devv env:bootstrap' to restore local sensitive files."
fi
info "Please restart your terminal or run 'exec zsh' to apply changes."
