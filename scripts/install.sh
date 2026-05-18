#!/usr/bin/env bash
# opendray universal installer dispatcher.
# Detects the host OS and runs the matching platform installer.
#
# Usage:
#   bash scripts/install.sh
#
# Or invoke a platform installer directly:
#   bash scripts/install-linux.sh        # Ubuntu / Debian
#   bash scripts/install-macos.sh        # macOS (Intel + Apple Silicon)
#   pwsh scripts/install-windows.ps1     # Windows (WSL2 setup helper)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

case "$(uname -s)" in
    Linux)
        # Inside WSL too — keep going, Linux installer handles WSL fine.
        exec bash "$SCRIPT_DIR/install-linux.sh" "$@"
        ;;
    Darwin)
        exec bash "$SCRIPT_DIR/install-macos.sh" "$@"
        ;;
    MINGW*|MSYS*|CYGWIN*)
        cat <<'EOF'
[!] Native Windows is not supported.
    opendray spawns PTY processes for AI CLIs; PTYs don't exist on
    native Windows, so the wizard can't proceed here.

    Recommended: install WSL2, then run install-linux.sh inside it.
    Run the PowerShell helper for WSL2 setup guidance:

        pwsh scripts/install-windows.ps1

EOF
        exit 1
        ;;
    *)
        echo "[!] Unsupported OS: $(uname -s)"
        echo "    Supported: Linux (Ubuntu/Debian), macOS."
        echo "    See docs/getting-started.md for manual install steps."
        exit 1
        ;;
esac
