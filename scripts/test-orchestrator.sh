#!/bin/bash
# Test Automation Orchestrator
# K-ERP SaaS Platform - Intelligent Test Execution Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_DIR="${PROJECT_ROOT}/tests/config"
RESULTS_DIR="${PROJECT_ROOT}/test-results"
COVERAGE_DIR="${PROJECT_ROOT}/coverage"
LOG_FILE="${RESULTS_DIR}/orchestrator.log"

# Default values
TEST_MODE="standard"
PARALLEL_WORKERS=4
FAIL_FAST=false
VERBOSE=false
DRY_RUN=false
COVERAGE=true
CHANGED_ONLY=false

# Timing
START_TIME=$(date +%s)

# ============================================================================
# Utility Functions
# ============================================================================

log() {
    local level=$1
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    case $level in
        INFO)  color=$BLUE ;;
        OK)    color=$GREEN ;;
        WARN)  color=$YELLOW ;;
        ERROR) color=$RED ;;
        *)     color=$NC ;;
    esac

    echo -e "${color}[${timestamp}] [${level}]${NC} ${message}"
    echo "[${timestamp}] [${level}] ${message}" >> "$LOG_FILE"
}

print_header() {
    echo ""
    echo -e "${CYAN}================================================================${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}================================================================${NC}"
    echo ""
}

print_section() {
    echo ""
    echo -e "${BLUE}--- $1 ---${NC}"
    echo ""
}

show_usage() {
    cat << EOF
Usage: $(basename $0) [OPTIONS] [TEST_TYPE...]

Test Automation Orchestrator for K-ERP SaaS Platform

TEST_TYPES:
    unit          Run unit tests
    integration   Run integration tests
    api           Run API tests
    e2e           Run end-to-end tests
    performance   Run performance tests
    all           Run all tests

OPTIONS:
    -m, --mode MODE       Execution mode: quick, standard, full, nightly (default: standard)
    -p, --parallel N      Number of parallel workers (default: 4)
    -f, --fail-fast       Stop on first failure
    -v, --verbose         Verbose output
    -c, --changed-only    Only test changed files
    --no-coverage         Disable coverage collection
    --dry-run             Show what would be run without executing
    -h, --help            Show this help message

EXAMPLES:
    $(basename $0) unit                    # Run unit tests
    $(basename $0) unit integration        # Run unit and integration tests
    $(basename $0) -m quick all            # Quick mode, all tests
    $(basename $0) -f --changed-only unit  # Fail-fast, only changed files

EOF
}

# ============================================================================
# Initialization
# ============================================================================

init() {
    mkdir -p "$RESULTS_DIR" "$COVERAGE_DIR"
    > "$LOG_FILE"

    log INFO "Initializing Test Orchestrator"
    log INFO "Project Root: $PROJECT_ROOT"
    log INFO "Test Mode: $TEST_MODE"
    log INFO "Parallel Workers: $PARALLEL_WORKERS"
}

# ============================================================================
# Test Discovery
# ============================================================================

discover_go_tests() {
    log INFO "Discovering Go tests..."

    local count=0
    local test_packages=()

    while IFS= read -r -d '' file; do
        local pkg_dir=$(dirname "$file")
        local pkg_path=${pkg_dir#$PROJECT_ROOT/}

        if [[ ! " ${test_packages[*]} " =~ " ${pkg_path} " ]]; then
            test_packages+=("$pkg_path")
            ((count++))
        fi
    done < <(find "$PROJECT_ROOT" -name "*_test.go" -not -path "*/vendor/*" -print0 2>/dev/null)

    log INFO "Found $count Go test packages"
    echo "${test_packages[@]}"
}

discover_python_tests() {
    log INFO "Discovering Python tests..."

    local count=0
    local test_files=()

    while IFS= read -r -d '' file; do
        test_files+=("$file")
        ((count++))
    done < <(find "$PROJECT_ROOT/python-services" -name "test_*.py" -o -name "*_test.py" -print0 2>/dev/null)

    log INFO "Found $count Python test files"
    echo "${test_files[@]}"
}

discover_frontend_tests() {
    log INFO "Discovering Frontend tests..."

    local count=0
    local test_files=()

    while IFS= read -r -d '' file; do
        test_files+=("$file")
        ((count++))
    done < <(find "$PROJECT_ROOT/web" -name "*.test.ts" -o -name "*.test.tsx" -o -name "*.spec.ts" -o -name "*.spec.tsx" -not -path "*/node_modules/*" -print0 2>/dev/null)

    log INFO "Found $count Frontend test files"
    echo "${test_files[@]}"
}

discover_e2e_tests() {
    log INFO "Discovering E2E tests..."

    local count=0
    local test_files=()

    while IFS= read -r -d '' file; do
        test_files+=("$file")
        ((count++))
    done < <(find "$PROJECT_ROOT/tests/e2e" -name "*.ts" -o -name "*.spec.ts" -print0 2>/dev/null)

    log INFO "Found $count E2E test files"
    echo "${test_files[@]}"
}

# ============================================================================
# Changed File Detection
# ============================================================================

get_changed_files() {
    if [[ "$CHANGED_ONLY" != "true" ]]; then
        return
    fi

    log INFO "Detecting changed files..."

    local base_branch="${BASE_BRANCH:-main}"
    local changed_files=$(git diff --name-only "$base_branch"...HEAD 2>/dev/null || git diff --name-only HEAD~1)

    log INFO "Changed files:"
    echo "$changed_files" | while read -r file; do
        log INFO "  - $file"
    done

    echo "$changed_files"
}

filter_tests_by_changes() {
    local changed_files="$1"
    local test_type="$2"

    # Simple mapping of paths to test types
    local run_tests=false

    echo "$changed_files" | while read -r file; do
        case "$file" in
            internal/domain/*|internal/service/*)
                [[ "$test_type" == "unit" || "$test_type" == "integration" ]] && run_tests=true
                ;;
            internal/handler/*|internal/middleware/*)
                [[ "$test_type" == "unit" || "$test_type" == "api" ]] && run_tests=true
                ;;
            web/*)
                [[ "$test_type" == "frontend" || "$test_type" == "e2e" ]] && run_tests=true
                ;;
            python-services/*)
                [[ "$test_type" == "python" ]] && run_tests=true
                ;;
            db/migrations/*)
                [[ "$test_type" == "integration" ]] && run_tests=true
                ;;
        esac
    done

    echo "$run_tests"
}

# ============================================================================
# Service Health Checks
# ============================================================================

check_postgres() {
    log INFO "Checking PostgreSQL..."

    if command -v pg_isready &> /dev/null; then
        if pg_isready -h localhost -p 5432 -q; then
            log OK "PostgreSQL is ready"
            return 0
        fi
    fi

    log WARN "PostgreSQL is not available"
    return 1
}

check_redis() {
    log INFO "Checking Redis..."

    if command -v redis-cli &> /dev/null; then
        if redis-cli -h localhost -p 6379 ping &> /dev/null; then
            log OK "Redis is ready"
            return 0
        fi
    fi

    log WARN "Redis is not available"
    return 1
}

check_nats() {
    log INFO "Checking NATS..."

    if nc -z localhost 4222 2>/dev/null; then
        log OK "NATS is ready"
        return 0
    fi

    log WARN "NATS is not available"
    return 1
}

check_services() {
    local services_ok=true

    print_section "Service Health Checks"

    check_postgres || services_ok=false
    check_redis || services_ok=false
    check_nats || services_ok=false

    if [[ "$services_ok" != "true" ]]; then
        log WARN "Some services are not available. Integration tests may fail."
    fi

    return 0
}

# ============================================================================
# Test Execution
# ============================================================================

run_go_unit_tests() {
    print_section "Go Unit Tests"

    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would run: go test -v -race -short ./..."
        return 0
    fi

    local coverage_flag=""
    if [[ "$COVERAGE" == "true" ]]; then
        coverage_flag="-coverprofile=${COVERAGE_DIR}/go-unit.out -covermode=atomic"
    fi

    local verbose_flag=""
    if [[ "$VERBOSE" == "true" ]]; then
        verbose_flag="-v"
    fi

    cd "$PROJECT_ROOT"

    if go test $verbose_flag -race -short -p $PARALLEL_WORKERS $coverage_flag ./... 2>&1 | tee "${RESULTS_DIR}/go-unit.log"; then
        log OK "Go unit tests passed"
        return 0
    else
        log ERROR "Go unit tests failed"
        return 1
    fi
}

run_go_integration_tests() {
    print_section "Go Integration Tests"

    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would run: go test -v -race -tags=integration ./..."
        return 0
    fi

    local coverage_flag=""
    if [[ "$COVERAGE" == "true" ]]; then
        coverage_flag="-coverprofile=${COVERAGE_DIR}/go-integration.out -covermode=atomic"
    fi

    cd "$PROJECT_ROOT"

    if go test -v -race -tags=integration -p $PARALLEL_WORKERS $coverage_flag ./... 2>&1 | tee "${RESULTS_DIR}/go-integration.log"; then
        log OK "Go integration tests passed"
        return 0
    else
        log ERROR "Go integration tests failed"
        return 1
    fi
}

run_python_tests() {
    print_section "Python Tests"

    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would run: pytest python-services/"
        return 0
    fi

    cd "$PROJECT_ROOT"

    local coverage_flag=""
    if [[ "$COVERAGE" == "true" ]]; then
        coverage_flag="--cov=python-services --cov-report=xml:${COVERAGE_DIR}/python.xml"
    fi

    local pytest_args="-n $PARALLEL_WORKERS --dist=loadfile"
    if [[ "$VERBOSE" == "true" ]]; then
        pytest_args="$pytest_args -v"
    fi

    if command -v pytest &> /dev/null; then
        if pytest $pytest_args $coverage_flag python-services/ 2>&1 | tee "${RESULTS_DIR}/python.log"; then
            log OK "Python tests passed"
            return 0
        else
            log ERROR "Python tests failed"
            return 1
        fi
    else
        log WARN "pytest not found, skipping Python tests"
        return 0
    fi
}

run_frontend_tests() {
    print_section "Frontend Tests"

    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would run: npm test in web/"
        return 0
    fi

    cd "$PROJECT_ROOT/web"

    if [[ -f "package.json" ]]; then
        if npm test 2>&1 | tee "${RESULTS_DIR}/frontend.log"; then
            log OK "Frontend tests passed"
            return 0
        else
            log ERROR "Frontend tests failed"
            return 1
        fi
    else
        log WARN "No package.json found in web/, skipping frontend tests"
        return 0
    fi
}

run_e2e_tests() {
    print_section "E2E Tests"

    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would run: playwright test"
        return 0
    fi

    cd "$PROJECT_ROOT"

    if command -v playwright &> /dev/null || npx playwright --version &> /dev/null; then
        if npx playwright test tests/e2e/ 2>&1 | tee "${RESULTS_DIR}/e2e.log"; then
            log OK "E2E tests passed"
            return 0
        else
            log ERROR "E2E tests failed"
            return 1
        fi
    else
        log WARN "Playwright not found, skipping E2E tests"
        return 0
    fi
}

run_performance_tests() {
    print_section "Performance Tests"

    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would run performance tests"
        return 0
    fi

    log WARN "Performance tests not yet implemented"
    return 0
}

# ============================================================================
# Coverage Report
# ============================================================================

generate_coverage_report() {
    print_section "Coverage Report"

    cd "$PROJECT_ROOT"

    # Merge Go coverage files
    if compgen -G "${COVERAGE_DIR}/go-*.out" > /dev/null; then
        log INFO "Merging Go coverage files..."

        # Simple concatenation (for atomic mode)
        echo "mode: atomic" > "${COVERAGE_DIR}/go-merged.out"
        tail -q -n +2 "${COVERAGE_DIR}/go-"*.out >> "${COVERAGE_DIR}/go-merged.out" 2>/dev/null || true

        # Generate HTML report
        if go tool cover -html="${COVERAGE_DIR}/go-merged.out" -o "${COVERAGE_DIR}/go-coverage.html" 2>/dev/null; then
            log OK "Go coverage report generated: ${COVERAGE_DIR}/go-coverage.html"
        fi

        # Calculate coverage percentage
        local coverage_pct=$(go tool cover -func="${COVERAGE_DIR}/go-merged.out" 2>/dev/null | grep total | awk '{print $3}')
        log INFO "Go Total Coverage: $coverage_pct"
    fi

    # Summary
    log INFO "Coverage reports available in: ${COVERAGE_DIR}/"
}

# ============================================================================
# Results Summary
# ============================================================================

print_results_summary() {
    local end_time=$(date +%s)
    local duration=$((end_time - START_TIME))

    print_header "Test Results Summary"

    echo -e "${CYAN}Duration:${NC} ${duration}s"
    echo ""

    # Count results
    local total=0
    local passed=0
    local failed=0

    for result in "${TEST_RESULTS[@]}"; do
        ((total++))
        if [[ "$result" == *":0" ]]; then
            ((passed++))
            echo -e "  ${GREEN}PASS${NC} ${result%:*}"
        else
            ((failed++))
            echo -e "  ${RED}FAIL${NC} ${result%:*}"
        fi
    done

    echo ""
    echo -e "${CYAN}Total:${NC} $total tests"
    echo -e "${GREEN}Passed:${NC} $passed"
    echo -e "${RED}Failed:${NC} $failed"
    echo ""

    if [[ $failed -gt 0 ]]; then
        echo -e "${RED}Some tests failed!${NC}"
        return 1
    else
        echo -e "${GREEN}All tests passed!${NC}"
        return 0
    fi
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    local test_types=()
    declare -a TEST_RESULTS=()

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -m|--mode)
                TEST_MODE="$2"
                shift 2
                ;;
            -p|--parallel)
                PARALLEL_WORKERS="$2"
                shift 2
                ;;
            -f|--fail-fast)
                FAIL_FAST=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -c|--changed-only)
                CHANGED_ONLY=true
                shift
                ;;
            --no-coverage)
                COVERAGE=false
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            all)
                test_types=("unit" "integration" "api" "e2e" "performance")
                shift
                ;;
            unit|integration|api|e2e|performance|frontend|python)
                test_types+=("$1")
                shift
                ;;
            *)
                echo "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    # Default to unit tests if no type specified
    if [[ ${#test_types[@]} -eq 0 ]]; then
        test_types=("unit")
    fi

    # Apply mode presets
    case $TEST_MODE in
        quick)
            PARALLEL_WORKERS=8
            COVERAGE=false
            ;;
        standard)
            PARALLEL_WORKERS=4
            ;;
        full)
            PARALLEL_WORKERS=4
            ;;
        nightly)
            PARALLEL_WORKERS=2
            test_types=("unit" "integration" "api" "e2e" "performance")
            ;;
    esac

    # Initialize
    init

    print_header "K-ERP Test Automation Orchestrator"
    log INFO "Mode: $TEST_MODE"
    log INFO "Test Types: ${test_types[*]}"
    log INFO "Parallel Workers: $PARALLEL_WORKERS"
    log INFO "Coverage: $COVERAGE"
    log INFO "Fail Fast: $FAIL_FAST"
    log INFO "Changed Only: $CHANGED_ONLY"
    log INFO "Dry Run: $DRY_RUN"

    # Check services
    check_services

    # Run tests
    for test_type in "${test_types[@]}"; do
        local result=0

        case $test_type in
            unit)
                run_go_unit_tests || result=1
                ;;
            integration)
                run_go_integration_tests || result=1
                ;;
            api)
                # API tests (same as integration for now)
                log INFO "API tests run as part of integration tests"
                ;;
            e2e)
                run_e2e_tests || result=1
                ;;
            frontend)
                run_frontend_tests || result=1
                ;;
            python)
                run_python_tests || result=1
                ;;
            performance)
                run_performance_tests || result=1
                ;;
        esac

        TEST_RESULTS+=("$test_type:$result")

        if [[ "$FAIL_FAST" == "true" && $result -ne 0 ]]; then
            log ERROR "Fail-fast triggered, stopping execution"
            break
        fi
    done

    # Generate coverage report
    if [[ "$COVERAGE" == "true" && "$DRY_RUN" != "true" ]]; then
        generate_coverage_report
    fi

    # Print summary
    print_results_summary
}

main "$@"
