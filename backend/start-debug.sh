#!/bin/bash

echo "Starting Backend Server in Debug Mode..."
echo "Delve debugger will listen on port 2345"
echo ""

cd "$(dirname "$0")"

# Find dlv in common locations
DLV_CMD=""
if command -v dlv &> /dev/null; then
    DLV_CMD="dlv"
else
    # Check common Go bin locations
    GOPATH_BIN="$HOME/go/bin/dlv"
    if [ -f "$GOPATH_BIN" ]; then
        DLV_CMD="$GOPATH_BIN"
    else
        # Try to get GOPATH from go env
        GOPATH=$(go env GOPATH 2>/dev/null)
        if [ -n "$GOPATH" ] && [ -f "$GOPATH/bin/dlv" ]; then
            DLV_CMD="$GOPATH/bin/dlv"
        else
            echo "Installing Delve debugger..."
            go install github.com/go-delve/delve/cmd/dlv@latest
            
            # Try again after installation
            GOPATH=$(go env GOPATH 2>/dev/null)
            if [ -n "$GOPATH" ] && [ -f "$GOPATH/bin/dlv" ]; then
                DLV_CMD="$GOPATH/bin/dlv"
                export PATH="$GOPATH/bin:$PATH"
            elif [ -f "$HOME/go/bin/dlv" ]; then
                DLV_CMD="$HOME/go/bin/dlv"
                export PATH="$HOME/go/bin:$PATH"
            fi
        fi
    fi
fi

if [ -z "$DLV_CMD" ]; then
    echo "Error: Could not find or install Delve debugger"
    echo "Please install manually: go install github.com/go-delve/delve/cmd/dlv@latest"
    echo "Then add \$GOPATH/bin to your PATH"
    exit 1
fi

echo "Using Delve: $DLV_CMD"
echo ""
echo "To attach VS Code debugger:"
echo "1. Open VS Code Debug panel (Cmd+Shift+D)"
echo "2. Select 'Attach to Debug Server' configuration"
echo "3. Press F5"
echo ""

# Start in debug mode
export DEBUG=true
$DLV_CMD debug --listen=:2345 --headless=true --api-version=2 --accept-multiclient cmd/server/main.go

