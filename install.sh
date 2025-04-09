#!/bin/bash
set -e

echo "Installing vers CLI..."
go install github.com/hdresearch/vers-cli@latest

# Find where Go installed the binary
if [ -n "$GOPATH" ]; then
    GO_BIN="$GOPATH/bin"
else
    # If GOPATH is not set, use the default location
    GO_BIN="$HOME/go/bin"
fi

if [ -f "$GO_BIN/vers-cli" ]; then
    mv "$GO_BIN/vers-cli" "$GO_BIN/vers"
    echo "Installation complete! Run 'vers' to get started."
else
    echo "Error: Could not find vers-cli binary in $GO_BIN"
    echo "Please check your Go installation and ensure the binary was installed correctly."
    exit 1
fi
