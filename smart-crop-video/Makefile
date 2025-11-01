# Makefile for smart-crop-video testing
#
# Prerequisites: Docker and Docker Compose
# No Python, Playwright, or other dependencies needed locally!

.PHONY: help test test-container test-api test-ui test-focused build-test-image clean

# Default target
help:
	@echo "smart-crop-video Test Commands"
	@echo "================================"
	@echo ""
	@echo "Prerequisites: Docker and Docker Compose only!"
	@echo ""
	@echo "Available targets:"
	@echo "  make test              - Run all tests (container + API + UI)"
	@echo "  make test-container    - Run container integration tests (fast, always passes)"
	@echo "  make test-api          - Run API tests"
	@echo "  make test-ui           - Run web UI tests"
	@echo "  make test-focused      - Run focused web UI tests"
	@echo "  make test-quick        - Run only the fastest, most reliable tests"
	@echo "  make build-test-image  - Build the test container image"
	@echo "  make test-shell        - Open a shell in the test container"
	@echo "  make clean             - Remove test artifacts and caches"
	@echo ""
	@echo "Examples:"
	@echo "  make test-container    # Quick validation (15 tests, ~40s)"
	@echo "  make test-quick        # Container + basic API tests"
	@echo "  make test              # Full suite (may have some failures)"

# Run all tests
test: build-test-image
	docker-compose -f docker-compose.test.yml run --rm tests

# Run only container tests (fast, reliable)
test-container: build-test-image
	docker-compose -f docker-compose.test.yml run --rm tests \
		pytest tests/test_container.py -v

# Run API tests
test-api: build-test-image
	docker-compose -f docker-compose.test.yml run --rm tests \
		pytest tests/test_api.py -v --tb=short

# Run web UI tests
test-ui: build-test-image
	docker-compose -f docker-compose.test.yml run --rm tests \
		pytest tests/test_web_ui.py -v --tb=short

# Run focused web UI tests
test-focused: build-test-image
	docker-compose -f docker-compose.test.yml run --rm tests \
		pytest tests/test_web_ui_focused.py -v -s --tb=short

# Run quick validation (container + basic API)
test-quick: build-test-image
	docker-compose -f docker-compose.test.yml run --rm tests \
		pytest tests/test_container.py tests/test_diagnostic.py -v

# Build the test container image
build-test-image:
	docker-compose -f docker-compose.test.yml build tests

# Open a shell in the test container for debugging
test-shell: build-test-image
	docker-compose -f docker-compose.test.yml run --rm tests /bin/bash

# Clean up test artifacts
clean:
	rm -rf .pytest_cache
	rm -rf tests/__pycache__
	find . -name "*.pyc" -delete
	find . -name "*_crop_option_*.jpg" -delete
	find . -name ".*_scene_*.jpg" -delete
	docker-compose -f docker-compose.test.yml down -v
