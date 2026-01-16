#!/bin/bash
# Pre-push hook for K-ERP
# Runs before git push to ensure code quality

set -e

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
    if grep -rn --include="*.go" --include="*.ts" --include="*.tsx" -E "(password|secret|api_key|apikey)\s*[:=]" . | grep -v "_test.go" | grep -v "\.example"; then
        echo "ERROR: Potential secrets found in code!"
        exit 1
    fi

    echo "[4/4] Building..."
    go build -o /dev/null ./cmd/api
fi

echo "=== Pre-Push Hook Passed ==="
