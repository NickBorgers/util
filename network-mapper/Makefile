BINARY_NAME=network-mapper
VERSION=1.0.0
BUILD_DIR=build
PLATFORMS=linux/amd64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all build clean test test-integration test-integration-quick test-integration-clean help

all: clean build

help:
	@echo "Available targets:"
	@echo "  build                  - Build binary for current platform"
	@echo "  build-all              - Build binaries for all platforms"
	@echo "  clean                  - Clean build directory"
	@echo "  test                   - Run unit tests"
	@echo "  test-integration       - Run integration tests with Docker"
	@echo "  test-integration-quick - Run integration tests without cleanup (for debugging)"
	@echo "  test-integration-clean - Clean up integration test containers"
	@echo "  run                    - Build and run the application"
	@echo "  deps                   - Download dependencies"

deps:
	go mod download
	go mod tidy

build: deps
	@echo "Building $(BINARY_NAME) for current platform..."
	# Build with full debug symbols and source information for transparency
	# This makes the binary fully debuggable and helps users understand how it works
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) .

build-all: clean deps
	@echo "Building $(BINARY_NAME) for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(BINARY_NAME)-$$GOOS-$$GOARCH; \
		if [ $$GOOS = "windows" ]; then output_name=$$output_name.exe; fi; \
		echo "Building for $$GOOS/$$GOARCH with full debug symbols..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$$output_name .; \
	done

clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)

test:
	@echo "Running unit tests..."
	go test -v ./...

test-integration:
	@echo "🧪 Running integration tests..."
	@docker compose -f docker-compose.test.yml up --build --abort-on-container-exit --exit-code-from test-runner
	@EXIT_CODE=$$?; \
	echo "🧹 Cleaning up test environment..."; \
	docker compose -f docker-compose.test.yml down -v; \
	if [ $$EXIT_CODE -eq 0 ]; then \
		echo "✅ Integration tests passed!"; \
	else \
		echo "❌ Integration tests failed!"; \
		exit $$EXIT_CODE; \
	fi

test-integration-quick:
	@echo "🧪 Running integration tests (quick mode - no cleanup)..."
	docker compose -f docker-compose.test.yml up --build --abort-on-container-exit --exit-code-from test-runner

test-integration-clean:
	@echo "🧹 Cleaning up test environment..."
	docker compose -f docker-compose.test.yml down -v

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .

# Development helpers
dev-deps:
	go install github.com/air-verse/air@latest

dev: dev-deps
	air

fmt:
	go fmt ./...

vet:
	go vet ./...

lint: vet fmt
	@echo "Code formatting and vetting complete"

# Set help as default target when running 'make' without arguments
.DEFAULT_GOAL := help