#!/bin/bash

# Exit on any error
set -e

echo "Installing GitAI..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go (https://go.dev/) or download a pre-built binary."
    exit 1
fi

# Create a temporary building directory
TEMP_BUILD_DIR=$(mktemp -d)
echo "Created temporary build directory: $TEMP_BUILD_DIR"

# Clone the repository to the temp directory
echo "Cloning codebase..."
git clone --depth 1 https://github.com/parthdande/gitai.git "$TEMP_BUILD_DIR"

# Build the binary inside the temp directory
echo "Compiling GitAI binary..."
(
    cd "$TEMP_BUILD_DIR"
    go build -o gitai cmd/main.go
)

# Install destination
DEST_DIR="/usr/local/bin"

echo "Moving binary to $DEST_DIR (requires sudo)..."
if sudo mv "$TEMP_BUILD_DIR/gitai" "$DEST_DIR/gitai"; then
    # Clean up the temp directory
    rm -rf "$TEMP_BUILD_DIR"
    echo "--------------------------------------------------------"
    echo " GitAI installed successfully to $DEST_DIR/gitai!"
    echo "--------------------------------------------------------"
    echo "Usage:"
    echo "  1. Set your API Key: export GEMINI_API_KEY=\"your-key\""
    echo "  2. Run 'gitai -commitmsg' or 'gitai -commit' in any Git repo"
    echo "--------------------------------------------------------"
else
    # Clean up the temp directory
    rm -rf "$TEMP_BUILD_DIR"
    echo "Failed to install binary to $DEST_DIR. Make sure you have sudo privileges."
    exit 1
fi
