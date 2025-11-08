#!/bin/bash
# Containerized test runner for smart-crop-video
#
# Prerequisites: Docker and Docker Compose only!
# No Python, Playwright, or other dependencies needed.
#
# Usage:
#   ./run-tests.sh                    # Run all tests (fast only, not comprehensive)
#   ./run-tests.sh fast               # Run fast tests only (unit + basic integration)
#   ./run-tests.sh comprehensive      # Run comprehensive end-to-end tests
#   ./run-tests.sh all-with-e2e       # Run ALL tests including comprehensive
#   ./run-tests.sh container          # Run container tests only
#   ./run-tests.sh quick              # Run quick validation
#   ./run-tests.sh shell              # Open test container shell

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print colored output
print_info() {
    echo -e "${GREEN}==>${NC} $1"
}

print_error() {
    echo -e "${RED}Error:${NC} $1"
}

print_usage() {
    echo "smart-crop-video Containerized Test Runner"
    echo "==========================================="
    echo ""
    echo "Prerequisites: Docker and Docker Compose only!"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  all              Run fast tests only (default) - unit, integration, API"
    echo "  fast             Run fast tests explicitly (same as 'all')"
    echo "  comprehensive    Run comprehensive end-to-end tests (10-15 min)"
    echo "  all-with-e2e     Run ALL tests including comprehensive"
    echo "  e2e              Run end-to-end video validation tests only"
    echo "  acceleration     Run acceleration feature tests only"
    echo "  container        Run container integration tests (fast, reliable)"
    echo "  api              Run API tests"
    echo "  ui               Run web UI tests"
    echo "  focused          Run focused web UI tests"
    echo "  quick            Run quick validation (container + diagnostic)"
    echo "  shell            Open a shell in the test container"
    echo "  build            Build the test container image"
    echo "  clean            Remove test artifacts"
    echo ""
    echo "Test Categories:"
    echo "  Fast (< 5 min):  Unit, integration, container, API tests"
    echo "  Comprehensive:   End-to-end crop validation + acceleration features"
    echo ""
    echo "Examples:"
    echo "  $0                  # Run fast tests (default)"
    echo "  $0 fast             # Run fast tests explicitly"
    echo "  $0 comprehensive    # Run comprehensive tests only"
    echo "  $0 all-with-e2e     # Run everything (15-20 min)"
    echo "  $0 e2e              # Test crop accuracy only"
    echo "  $0 quick            # Fastest smoke test"
}

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed or not in PATH"
    echo "Please install Docker: https://docs.docker.com/get-docker/"
    exit 1
fi

# Check if Docker Compose is available
if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed or not in PATH"
    echo "Please install Docker Compose: https://docs.docker.com/compose/install/"
    exit 1
fi

# Get command (default to 'all')
COMMAND=${1:-all}

# Change to script directory
cd "$(dirname "$0")"

case "$COMMAND" in
    all)
        print_info "Running fast tests (unit, integration, API)..."
        print_info "Note: Use 'all-with-e2e' to include comprehensive tests"
        docker-compose -f docker-compose.test.yml build tests
        docker-compose -f docker-compose.test.yml run --rm tests \
            pytest tests/ -m "not comprehensive" -v
        ;;

    fast)
        print_info "Running fast tests explicitly..."
        docker-compose -f docker-compose.test.yml build tests
        docker-compose -f docker-compose.test.yml run --rm tests \
            pytest tests/ -m "not comprehensive" -v
        ;;

    comprehensive)
        print_info "Running comprehensive end-to-end tests..."
        print_info "This will take 10-15 minutes..."
        docker-compose -f docker-compose.test.yml build tests
        docker-compose -f docker-compose.test.yml run --rm tests \
            pytest tests/integration/test_end_to_end_video.py tests/integration/test_acceleration.py \
            -m comprehensive -v --tb=short
        ;;

    all-with-e2e)
        print_info "Running ALL tests including comprehensive..."
        print_info "This will take 15-20 minutes..."
        docker-compose -f docker-compose.test.yml build tests
        docker-compose -f docker-compose.test.yml run --rm tests \
            pytest tests/ -v --tb=short
        ;;

    e2e)
        print_info "Running end-to-end video validation tests only..."
        docker-compose -f docker-compose.test.yml build tests
        docker-compose -f docker-compose.test.yml run --rm tests \
            pytest tests/integration/test_end_to_end_video.py -m comprehensive -v --tb=short
        ;;

    acceleration)
        print_info "Running acceleration feature tests only..."
        docker-compose -f docker-compose.test.yml build tests
        docker-compose -f docker-compose.test.yml run --rm tests \
            pytest tests/integration/test_acceleration.py -m comprehensive -v --tb=short
        ;;

    container)
        print_info "Running container integration tests..."
        docker-compose -f docker-compose.test.yml build tests
        docker-compose -f docker-compose.test.yml run --rm tests \
            pytest tests/test_container.py -v
        ;;

    api)
        print_info "Running API tests..."
        docker-compose -f docker-compose.test.yml build tests
        docker-compose -f docker-compose.test.yml run --rm tests \
            pytest tests/test_api.py -v --tb=short
        ;;

    ui)
        print_info "Running web UI tests..."
        docker-compose -f docker-compose.test.yml build tests
        docker-compose -f docker-compose.test.yml run --rm tests \
            pytest tests/test_web_ui.py -v --tb=short
        ;;

    focused)
        print_info "Running focused web UI tests..."
        docker-compose -f docker-compose.test.yml build tests
        docker-compose -f docker-compose.test.yml run --rm tests \
            pytest tests/test_web_ui_focused.py -v -s --tb=short
        ;;

    quick)
        print_info "Running quick validation tests..."
        docker-compose -f docker-compose.test.yml build tests
        docker-compose -f docker-compose.test.yml run --rm tests \
            pytest tests/test_container.py tests/test_diagnostic.py -v
        ;;

    shell)
        print_info "Opening shell in test container..."
        docker-compose -f docker-compose.test.yml build tests
        docker-compose -f docker-compose.test.yml run --rm tests /bin/bash
        ;;

    build)
        print_info "Building test container image..."
        docker-compose -f docker-compose.test.yml build tests
        ;;

    clean)
        print_info "Cleaning test artifacts..."
        rm -rf .pytest_cache
        rm -rf tests/__pycache__
        find . -name "*.pyc" -delete 2>/dev/null || true
        find . -name "*_crop_option_*.jpg" -delete 2>/dev/null || true
        find . -name ".*_scene_*.jpg" -delete 2>/dev/null || true
        docker-compose -f docker-compose.test.yml down -v
        print_info "Cleanup complete"
        ;;

    help|--help|-h)
        print_usage
        ;;

    *)
        print_error "Unknown command: $COMMAND"
        echo ""
        print_usage
        exit 1
        ;;
esac
