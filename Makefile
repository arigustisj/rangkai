.PHONY: build test lint clean install uninstall verify

BINARY=rangkai
BUILD_DIR=bin
INSTALL_PATH=/usr/local/bin/$(BINARY)

build:
	@echo "Building $(BINARY)..."
	@go build -o $(BUILD_DIR)/$(BINARY) ./cmd/rangkai

test:
	@go test ./...

lint:
	@golangci-lint run ./...

clean:
	@rm -rf $(BUILD_DIR)

install: build
	@echo "Installing to $(INSTALL_PATH)..."
	@sudo cp $(BUILD_DIR)/$(BINARY) $(INSTALL_PATH)

uninstall:
	@echo "Removing $(INSTALL_PATH)..."
	@sudo rm -f $(INSTALL_PATH)

verify: build
	@$(BUILD_DIR)/$(BINARY) verify

run: build
	@$(BUILD_DIR)/$(BINARY)

dev:
	@go run ./cmd/rangkai

help:
	@echo "Available targets:"
	@echo "  build     - Build binary to bin/"
	@echo "  test      - Run tests"
	@echo "  lint      - Run linter"
	@echo "  clean     - Remove build artifacts"
	@echo "  install   - Install to /usr/local/bin"
	@echo "  uninstall - Remove from /usr/local/bin"
	@echo "  verify    - Build and run verify"
	@echo "  run       - Build and run"
	@echo "  dev       - Run with go run"