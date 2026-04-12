#!/usr/bin/env bash
# oauth4os-demo installer
# Usage: curl -sL https://f5cmk2hxwx.us-west-2.awsapprunner.com/install.sh | bash
set -euo pipefail

PROXY="${OAUTH4OS_URL:-https://f5cmk2hxwx.us-west-2.awsapprunner.com}"
INSTALL_DIR="${HOME}/.local/bin"
SCRIPT_NAME="oauth4os-demo"

echo "🔐 Installing oauth4os-demo CLI..."

mkdir -p "$INSTALL_DIR"

curl -sf "${PROXY}/scripts/oauth4os-demo" -o "${INSTALL_DIR}/${SCRIPT_NAME}" || {
  echo "❌ Download failed. Is the proxy running at ${PROXY}?"
  exit 1
}
chmod +x "${INSTALL_DIR}/${SCRIPT_NAME}"

# Check PATH
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  echo ""
  echo "⚠️  Add to your PATH:"
  echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
  echo ""
  echo "Or add to your shell profile:"
  echo "  echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.bashrc"
fi

echo ""
echo "✅ Installed to ${INSTALL_DIR}/${SCRIPT_NAME}"
echo ""
echo "Get started:"
echo "  oauth4os-demo health     # check proxy"
echo "  oauth4os-demo login      # authenticate"
echo "  oauth4os-demo search 'level:ERROR'"
echo "  oauth4os-demo services   # list services"
echo "  oauth4os-demo tail       # latest logs"
