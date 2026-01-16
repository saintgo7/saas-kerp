#!/bin/bash
# Post-commit hook for K-ERP
# Runs after each commit for logging and notifications

COMMIT_MSG=$(git log -1 --pretty=%B)
COMMIT_HASH=$(git rev-parse --short HEAD)
BRANCH=$(git branch --show-current)
AUTHOR=$(git log -1 --pretty=%an)

echo "Commit: $COMMIT_HASH on $BRANCH by $AUTHOR"
echo "Message: $COMMIT_MSG"

# Log to dev-log if on feature branch
if [[ "$BRANCH" == feature/* ]]; then
    LOG_FILE="docs/dev-log/commits.log"
    echo "[$(date +%Y-%m-%d\ %H:%M:%S)] $COMMIT_HASH - $COMMIT_MSG" >> "$LOG_FILE" 2>/dev/null || true
fi

# Notify if pushing to develop (optional Slack webhook)
if [[ "$BRANCH" == "develop" && -n "$SLACK_WEBHOOK" ]]; then
    curl -s -X POST "$SLACK_WEBHOOK" \
        -H "Content-Type: application/json" \
        -d "{\"text\": \"New commit on develop: $COMMIT_MSG\"}" \
        > /dev/null 2>&1 || true
fi
