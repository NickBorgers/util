#!/bin/bash
# Run smart-crop-video tests in Docker container
# This allows running all tests (including integration tests) without installing dependencies locally

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

IMAGE_NAME="smart-crop-video:test"

# Function to print colored output
print_info() {
    echo -e "${GREEN}==>${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}==>${NC} $1"
}

print_error() {
    echo -e "${RED}==>${NC} $1"
}

# Function to build the test image
build_image() {
    print_info "Building test Docker image..."
    docker build -f Dockerfile.test -t "$IMAGE_NAME" .
    print_info "Test image built successfully!"
}

# Function to run tests
run_tests() {
    local test_args="$@"

    print_info "Running tests in Docker container..."
    docker run --rm \
        -v "$(pwd)/tests:/app/tests" \
        "$IMAGE_NAME" \
        pytest $test_args
}

# Parse command line arguments
case "${1:-all}" in
    build)
        build_image
        ;;

    unit)
        print_info "Running unit tests only..."
        run_tests "tests/unit/" "-v"
        ;;

    integration)
        print_info "Running integration tests only (with FFmpeg)..."
        run_tests "tests/integration/" "-v"
        ;;

    all)
        print_info "Running containerized tests (unit + integration)..."
        print_warning "Note: Container/API/UI tests are skipped (they require Docker-in-Docker)"
        run_tests "tests/unit/" "tests/integration/" "-v"
        ;;

    quick)
        print_info "Running quick unit tests..."
        run_tests "tests/unit/" "-q"
        ;;

    coverage)
        print_info "Running tests with coverage report..."
        run_tests "tests/unit/" "-v" "--cov=smart_crop" "--cov-report=term-missing"
        ;;

    shell)
        print_info "Opening shell in test container..."
        docker run --rm -it \
            -v "$(pwd)/tests:/app/tests" \
            "$IMAGE_NAME" \
            /bin/sh
        ;;

    help|--help|-h)
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  build        Build the test Docker image"
        echo "  unit         Run unit tests only (286 tests, ~0.2s)"
        echo "  integration  Run integration tests with FFmpeg (8 tests, ~2s)"
        echo "  all          Run unit + integration tests (294 tests, default)"
        echo "  quick        Run unit tests with minimal output"
        echo "  coverage     Run tests with coverage report"
        echo "  shell        Open interactive shell in test container"
        echo "  help         Show this help message"
        echo ""
        echo "Note: Container/API/UI tests (34 tests) cannot run in Docker"
        echo "      They require Docker-in-Docker. Run these on the host:"
        echo "      pytest tests/test_container.py tests/test_api.py -v"
        echo ""
        echo "Examples:"
        echo "  $0 build              # Build the test image first"
        echo "  $0 unit               # Run 286 unit tests"
        echo "  $0 integration        # Run 8 integration tests with FFmpeg"
        echo "  $0 all                # Run 294 containerized tests"
        echo "  $0 shell              # Debug in container"
        ;;

    *)
        print_error "Unknown command: $1"
        echo "Run '$0 help' for usage information"
        exit 1
        ;;
esac
