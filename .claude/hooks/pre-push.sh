#!/bin/bash
# Pre-push hook for K-ERP
# Runs before git push to ensure code quality

set -euo pipefail

echo "=== K-ERP Pre-Push Hook ==="

# Get current branch
BRANCH=$(git branch --show-current)

# Only run full checks on develop and main
if [[ "$BRANCH" == "develop" || "$BRANCH" == "main" ]]; then
    echo "[1/4] Running Go linter..."
    if command -v golangci-lint &> /dev/null; then
        golangci-lint run ./... --timeout=5m
    else
        echo "Warning: golangci-lint not found, skipping..."
    fi

    echo "[2/4] Running Go tests..."
    go test -race -short ./...

    echo "[3/4] Checking for secrets..."
    # The previous check excluded *.example files, which is exactly where
    # deployment credentials get pasted - and why the production SSH password
    # in docs/.env_ssh.example was never caught. scan-secrets.py reads those
    # files and judges the VALUE instead.
    python3 "$(git rev-parse --show-toplevel)/scripts/scan-secrets.py"

    echo "[4/4] Building..."
    go build -o /dev/null ./cmd/api
fi

echo "=== Pre-Push Hook Passed ==="
