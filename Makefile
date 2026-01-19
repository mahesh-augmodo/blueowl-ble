# Variables
BINARY_NAME=blueowl-ble
CMD_DIR=./cmd/server
TEST_RECORDINGS_DIR=./test_recordings

# Default target: Build the binary
.PHONY: all
all: build

# Build for the current architecture (Mac/Windows/Linux)
.PHONY: build
build:
	@echo "🔨 Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete"

# Build specifically for Raspberry Pi (Linux/ARM64)
# Useful if you want to compile on Mac and SCP to the Pi later
.PHONY: build-pi
build-pi:
	@echo "Building for Raspberry Pi (Linux ARM64)..."
	GOOS=linux GOARCH=arm64 go build -o $(BINARY_NAME)-pi $(CMD_DIR)
	@echo "Pi Build complete: $(BINARY_NAME)-pi"

# Build and Run immediately
.PHONY: run
run: build
	@echo "🚀 Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Clean up binaries and mock data
.PHONY: clean
clean:
	@echo "🧹 Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-pi
	rm -rf $(TEST_RECORDINGS_DIR)
	@echo "Cleaned."