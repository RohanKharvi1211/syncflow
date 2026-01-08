# Starting Server in Debug Mode

## Option 1: VS Code Debug Panel (Easiest - Recommended)

1. **Open VS Code**
2. **Press `Cmd+Shift+D`** (or `Ctrl+Shift+D` on Windows/Linux) to open Debug panel
3. **Select "Launch Backend Server"** from the dropdown
4. **Press `F5`** or click the green play button
5. The server starts in debug mode with breakpoints enabled!

## Option 2: Command Line with Delve (Headless Mode)

Run this script:
```bash
cd backend
./start-debug.sh
```

Or manually:
```bash
cd backend
dlv debug --listen=:2345 --headless=true --api-version=2 --accept-multiclient cmd/server/main.go
```

Then in VS Code:
1. Select "Attach to Debug Server" configuration
2. Press F5

## Option 3: Direct Delve (Interactive)

```bash
cd backend
dlv debug cmd/server/main.go
```

This opens an interactive debugger console. Commands:
- `break main.main` - Set breakpoint
- `continue` or `c` - Continue execution
- `next` or `n` - Step over
- `step` or `s` - Step into
- `print variableName` - Print variable
- `exit` or `quit` - Exit debugger

## Option 4: Environment Variable (Debug Logging)

For debug logging (not full debugger, but useful):
```bash
cd backend
DEBUG=true go run cmd/server/main.go
```

This enables debug logging from the `utils.Debug()` functions.

## Current Status

To check if debug server is running:
```bash
lsof -i :2345
```

To stop any running server:
```bash
pkill -f "dlv\|main.go"
```

## Quick Start (VS Code)

**Just press `F5` in VS Code with "Launch Backend Server" selected!**

This is the easiest way - VS Code handles everything automatically.


