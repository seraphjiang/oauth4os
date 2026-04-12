#!/usr/bin/env bash
# oauth4os-demo installer
# Usage: curl -sL https://f5cmk2hxwx.us-west-2.awsapprunner.com/install.sh | bash
set -euo pipefail

PROXY="${OAUTH4OS_PROXY:-https://f5cmk2hxwx.us-west-2.awsapprunner.com}"
INSTALL_DIR="${HOME}/.local/bin"
SCRIPT_NAME="oauth4os-demo"

RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

echo -e "${BOLD}oauth4os-demo installer${NC}"
echo ""

# Check deps
for cmd in curl jq openssl; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo -e "${RED}Error: $cmd is required. Install it first.${NC}"
    exit 1
  fi
done

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux|darwin) ;;
  *) echo -e "${RED}Unsupported OS: $OS${NC}"; exit 1 ;;
esac

# Create install dir
mkdir -p "$INSTALL_DIR"

# Download CLI wrapper
echo -e "${CYAN}Downloading ${SCRIPT_NAME}...${NC}"
curl -sfL "${PROXY}/scripts/oauth4os-demo" -o "${INSTALL_DIR}/${SCRIPT_NAME}"
chmod +x "${INSTALL_DIR}/${SCRIPT_NAME}"

# Check PATH
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  SHELL_NAME=$(basename "$SHELL")
  case "$SHELL_NAME" in
    zsh)  RC="$HOME/.zshrc" ;;
    bash) RC="$HOME/.bashrc" ;;
    *)    RC="$HOME/.profile" ;;
  esac
  echo "export PATH=\"${INSTALL_DIR}:\$PATH\"" >> "$RC"
  echo -e "${CYAN}Added ${INSTALL_DIR} to PATH in ${RC}${NC}"
  export PATH="${INSTALL_DIR}:$PATH"
fi

echo ""
echo -e "${GREEN}✅ Installed ${SCRIPT_NAME} to ${INSTALL_DIR}/${SCRIPT_NAME}${NC}"
echo ""
echo -e "${BOLD}Quick start:${NC}"
echo "  oauth4os-demo status    # Check proxy health"
echo "  oauth4os-demo login     # Authenticate via browser"
echo "  oauth4os-demo search 'level:ERROR'"
echo "  oauth4os-demo services  # List indexed services"
echo ""
