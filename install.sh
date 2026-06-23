#!/bin/bash

# Exit on any error
set -e

if [ "$GITAI_UPDATE" = "true" ]; then
    echo "Updating GitAI..."
else
    echo "Installing GitAI..."
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go (https://go.dev/)."
    exit 1
fi

# Create a temporary building directory
TEMP_BUILD_DIR=$(mktemp -d)

# Clone the repository to the temp directory silently
git clone -q --depth 1 https://github.com/parth-nformis/gitai.git "$TEMP_BUILD_DIR" 2>/dev/null

# Build the binary inside the temp directory
(
    cd "$TEMP_BUILD_DIR"
    go build -o gitai cmd/main.go
)

# Install destination
DEST_DIR="/usr/local/bin"

# Create config directory and default config file
CONFIG_DIR="$HOME/.gitai"
mkdir -p "$CONFIG_DIR"

if [ ! -f "$CONFIG_DIR/gitai.json" ]; then
    cat > "$CONFIG_DIR/gitai.json" << 'EOF'
{
  "api_base": "",
  "api_key": "",
  "model": ""
}
EOF
    echo " Created config file at $CONFIG_DIR/gitai.json"
fi

if sudo mv "$TEMP_BUILD_DIR/gitai" "$DEST_DIR/gitai"; then
    # Clean up the temp directory
    rm -rf "$TEMP_BUILD_DIR"
    echo "--------------------------------------------------------"
    if [ "$GITAI_UPDATE" = "true" ]; then
        echo " GitAI updated successfully!"
    else
        echo " GitAI installed successfully!"
    fi
    echo "--------------------------------------------------------"
    if [ "$GITAI_UPDATE" != "true" ]; then
        echo "Next steps:"
        echo "  1. Add your API key to $CONFIG_DIR/gitai.json"
        echo "     (or set GEMINI_API_KEY environment variable)"
        echo "  2. Run 'gitai -commitmsg' or 'gitai -commit' in any Git repo"
        echo "--------------------------------------------------------"
    fi
else
    # Clean up the temp directory
    rm -rf "$TEMP_BUILD_DIR"
    echo "Failed to install binary to $DEST_DIR. Make sure you have sudo privileges."
    exit 1
fi

