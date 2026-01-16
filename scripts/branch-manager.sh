#!/usr/bin/env bash
# K-ERP Branch Manager (macOS compatible)
# Usage: ./scripts/branch-manager.sh [command] [phase]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Phase branch mapping (compatible with bash 3.x)
get_branch() {
    case "$1" in
        1) echo "phase/1-infra" ;;
        2) echo "phase/2-database" ;;
        3) echo "phase/3-go-core" ;;
        4) echo "phase/4-go-biz" ;;
        5) echo "phase/5-tax" ;;
        6) echo "phase/6-edi" ;;
        7) echo "phase/7-frontend" ;;
        8) echo "phase/8-testing" ;;
        *) echo "" ;;
    esac
}

get_name() {
    case "$1" in
        1) echo "Infrastructure" ;;
        2) echo "Database" ;;
        3) echo "Go Core" ;;
        4) echo "Go Business" ;;
        5) echo "Tax Scraper" ;;
        6) echo "Insurance EDI" ;;
        7) echo "Frontend" ;;
        8) echo "Testing" ;;
        *) echo "" ;;
    esac
}

show_help() {
    cat << 'EOF'
K-ERP Branch Manager
====================

Usage: ./scripts/branch-manager.sh [command] [phase]

Commands:
  status              Show all branches status
  switch <phase>      Switch to phase branch (1-8)
  update <phase>      Update phase branch from develop
  merge <phase>       Merge phase branch to develop
  sync                Sync all phase branches with develop
  create              Create all phase branches (if not exist)

Examples:
  ./scripts/branch-manager.sh status
  ./scripts/branch-manager.sh switch 3
  ./scripts/branch-manager.sh merge 1
  ./scripts/branch-manager.sh update 5

Branch Structure:
  main                Production releases
  develop             Integration branch
  phase/1-infra       Infrastructure & DevOps
  phase/2-database    Database & Schema
  phase/3-go-core     Go Core API
  phase/4-go-biz      Go Business Modules
  phase/5-tax         Python Tax Scraper
  phase/6-edi         Python Insurance EDI
  phase/7-frontend    Frontend
  phase/8-testing     Integration & Testing
EOF
}

show_status() {
    echo "================================================================================"
    echo "                    K-ERP Branch Status"
    echo "================================================================================"
    echo ""

    local current=$(git branch --show-current)
    echo "Current Branch: $current"
    echo ""

    printf "%-20s | %-5s | %-11s | %s\n" "Branch" "Phase" "Status" "Last Commit"
    echo "------------------|-------|-------------|----------------------------------"

    for i in 1 2 3 4 5 6 7 8; do
        local branch=$(get_branch $i)
        local name=$(get_name $i)

        if git show-ref --verify --quiet "refs/heads/$branch"; then
            local last_commit=$(git log -1 --format="%h %s" "$branch" 2>/dev/null | cut -c1-35)
            local ahead=$(git rev-list --count "develop..$branch" 2>/dev/null || echo "0")
            local behind=$(git rev-list --count "$branch..develop" 2>/dev/null || echo "0")
            local status="+$ahead/-$behind"

            if [[ "$branch" == "$current" ]]; then
                printf "* %-18s | %-5s | %-11s | %s\n" "$branch" "$i" "$status" "$last_commit"
            else
                printf "  %-18s | %-5s | %-11s | %s\n" "$branch" "$i" "$status" "$last_commit"
            fi
        else
            printf "  %-18s | %-5s | %-11s | %s\n" "$branch" "$i" "NOT CREATED" ""
        fi
    done

    echo "------------------|-------|-------------|----------------------------------"
    echo ""
    echo "Legend: +ahead/-behind from develop"
}

switch_branch() {
    local phase=$1
    local branch=$(get_branch "$phase")

    if [[ -z "$branch" ]]; then
        echo "Error: Invalid phase number. Use 1-8."
        exit 1
    fi

    echo "Switching to $branch (Phase $phase: $(get_name $phase))..."

    # Check for uncommitted changes
    if ! git diff --quiet || ! git diff --cached --quiet; then
        echo "Warning: You have uncommitted changes."
        read -p "Stash changes and continue? [y/N] " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            git stash push -m "Auto-stash before switching to $branch"
        else
            echo "Aborted."
            exit 1
        fi
    fi

    git checkout "$branch"
    echo "Switched to $branch"
}

update_branch() {
    local phase=$1
    local branch=$(get_branch "$phase")

    if [[ -z "$branch" ]]; then
        echo "Error: Invalid phase number. Use 1-8."
        exit 1
    fi

    echo "Updating $branch from develop..."

    local current=$(git branch --show-current)

    git checkout "$branch"
    git fetch origin develop 2>/dev/null || true
    git rebase develop || {
        echo "Rebase conflict! Resolve manually or run: git rebase --abort"
        exit 1
    }

    echo "Updated $branch successfully."

    if [[ "$current" != "$branch" ]]; then
        git checkout "$current"
    fi
}

merge_branch() {
    local phase=$1
    local branch=$(get_branch "$phase")

    if [[ -z "$branch" ]]; then
        echo "Error: Invalid phase number. Use 1-8."
        exit 1
    fi

    echo "Merging $branch into develop..."

    local current=$(git branch --show-current)

    # Switch to develop
    git checkout develop
    git pull origin develop 2>/dev/null || true

    # Merge phase branch
    git merge "$branch" --no-ff -m "Merge $branch into develop (Phase $phase complete)"

    echo "Merged $branch into develop."
    echo "Don't forget to push: git push origin develop"

    # Return to original branch
    if [[ "$current" != "develop" ]]; then
        git checkout "$current"
    fi
}

sync_all() {
    echo "Syncing all phase branches with develop..."

    git fetch origin develop 2>/dev/null || true

    for i in 1 2 3 4 5 6 7 8; do
        local branch=$(get_branch $i)

        if git show-ref --verify --quiet "refs/heads/$branch"; then
            echo ""
            echo "--- Updating $branch ---"
            git checkout "$branch"
            git rebase develop || {
                echo "Conflict in $branch. Skipping..."
                git rebase --abort
            }
        fi
    done

    git checkout develop
    echo ""
    echo "Sync complete."
}

create_branches() {
    echo "Creating phase branches..."

    # Ensure develop exists
    if ! git show-ref --verify --quiet "refs/heads/develop"; then
        git checkout -b develop
    fi

    for i in 1 2 3 4 5 6 7 8; do
        local branch=$(get_branch $i)

        if git show-ref --verify --quiet "refs/heads/$branch"; then
            echo "  $branch already exists"
        else
            git checkout -b "$branch" develop
            echo "  Created $branch"
        fi
    done

    git checkout develop
    echo "All branches created."
}

# Main
case "${1:-}" in
    "status"|"")
        show_status
        ;;
    "switch")
        switch_branch "$2"
        ;;
    "update")
        update_branch "$2"
        ;;
    "merge")
        merge_branch "$2"
        ;;
    "sync")
        sync_all
        ;;
    "create")
        create_branches
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        echo "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
