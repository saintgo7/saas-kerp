#!/bin/bash
# K-ERP SaaS 8-Phase Parallel Development Launcher
# Run: ./scripts/start-dev.sh [phase]
#
# Usage:
#   ./scripts/start-dev.sh        # Show guide
#   ./scripts/start-dev.sh 1      # Open Terminal for Phase 1
#   ./scripts/start-dev.sh all    # Open all 8 terminals (macOS only)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

show_guide() {
    cat << 'EOF'
================================================================================
                    K-ERP SaaS v0.2 - Development Setup
================================================================================

8 PHASE PARALLEL DEVELOPMENT
--------------------------------------------------------------------------------
Phase | Agent  | Focus                          | Command
------|--------|--------------------------------|---------------------------
  1   | @ops   | Docker, CI/CD, Nginx, SSL      | ./scripts/start-dev.sh 1
  2   | @db    | PostgreSQL, Migrations, RLS    | ./scripts/start-dev.sh 2
  3   | @go    | Router, Middleware, Auth       | ./scripts/start-dev.sh 3
  4   | @acc   | Accounting, Voucher, Invoice   | ./scripts/start-dev.sh 4
  5   | @tax   | Hometax, gRPC, SEED            | ./scripts/start-dev.sh 5
  6   | @hr    | 4대보험, EDI, ARIA             | ./scripts/start-dev.sh 6
  7   | @react | React, TypeScript, Vite        | ./scripts/start-dev.sh 7
  8   | test   | Unit, E2E, Security            | ./scripts/start-dev.sh 8
--------------------------------------------------------------------------------

QUICK START
--------------------------------------------------------------------------------
1. Open 8 terminal tabs/windows
2. In each terminal, run: cd ~/01_DEV/SaaS_erp_clone-260116 && claude
3. Once Claude starts, type: /saas {phase_number}

Or use this script:
  ./scripts/start-dev.sh all    # Opens all 8 terminals (macOS)
--------------------------------------------------------------------------------

AVAILABLE SKILLS (inside Claude)
--------------------------------------------------------------------------------
/saas          - Main dashboard
/saas 1-8      - Enter specific phase
/api {name}    - Go REST API scaffolding
/grpc {name}   - gRPC Proto generation
/db {name}     - Database migration
/py {name}     - Python gRPC service
/test {name}   - Test code generation
/deploy        - Deploy to remote server
/status        - Check development status
--------------------------------------------------------------------------------
EOF
}

start_phase() {
    local phase=$1
    local phase_names=(
        ""
        "Infrastructure (@ops)"
        "Database (@db)"
        "Go Core (@go)"
        "Go Business (@acc)"
        "Tax Scraper (@tax)"
        "Insurance EDI (@hr)"
        "Frontend (@react)"
        "Testing"
    )

    echo "Starting Phase $phase: ${phase_names[$phase]}"
    echo "---"
    echo "cd $PROJECT_ROOT && claude"
    echo ""
    echo "Once Claude starts, type: /saas $phase"
}

start_all_macos() {
    echo "Opening 8 terminal windows for K-ERP development..."

    for i in {1..8}; do
        osascript -e "
            tell application \"Terminal\"
                activate
                do script \"cd '$PROJECT_ROOT' && echo 'Phase $i - Type: /saas $i after starting claude' && claude\"
            end tell
        " 2>/dev/null || {
            echo "Error: Could not open Terminal for Phase $i"
            echo "Please open terminals manually and run: cd $PROJECT_ROOT && claude"
            exit 1
        }
        sleep 0.5
    done

    echo "8 terminals opened. Type '/saas {phase}' in each Claude session."
}

# Main
case "${1:-}" in
    "")
        show_guide
        ;;
    "all")
        if [[ "$OSTYPE" == "darwin"* ]]; then
            start_all_macos
        else
            echo "Auto-open only supported on macOS."
            echo "Please open 8 terminals manually."
            show_guide
        fi
        ;;
    [1-8])
        start_phase "$1"
        ;;
    *)
        echo "Usage: $0 [phase|all]"
        echo "  phase: 1-8"
        echo "  all: Open all 8 terminals (macOS only)"
        exit 1
        ;;
esac
