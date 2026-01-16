#!/bin/bash
# Install git hooks for K-ERP development
# Run: ./scripts/install-hooks.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
HOOKS_DIR="$PROJECT_ROOT/.claude/hooks"
GIT_HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

echo "Installing K-ERP Git Hooks..."

# Create .git/hooks if not exists
mkdir -p "$GIT_HOOKS_DIR"

# Install pre-push hook
if [ -f "$HOOKS_DIR/pre-push.sh" ]; then
    cp "$HOOKS_DIR/pre-push.sh" "$GIT_HOOKS_DIR/pre-push"
    chmod +x "$GIT_HOOKS_DIR/pre-push"
    echo "[OK] pre-push hook installed"
else
    echo "[SKIP] pre-push.sh not found"
fi

# Install post-commit hook
if [ -f "$HOOKS_DIR/post-commit.sh" ]; then
    cp "$HOOKS_DIR/post-commit.sh" "$GIT_HOOKS_DIR/post-commit"
    chmod +x "$GIT_HOOKS_DIR/post-commit"
    echo "[OK] post-commit hook installed"
else
    echo "[SKIP] post-commit.sh not found"
fi

echo ""
echo "Git hooks installed successfully!"
echo "Hooks directory: $GIT_HOOKS_DIR"
