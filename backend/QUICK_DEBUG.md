# Quick Debug Start

## ✅ Delve is Installed!

Delve is installed at: `/Users/rohan/go/bin/dlv`

## Start Debug Server

### Option 1: Use the Script (Fixed)
```bash
cd backend
export PATH="$HOME/go/bin:$PATH"  # Add to PATH for this session
./start-debug.sh
```

### Option 2: Direct Command
```bash
cd backend
export PATH="$HOME/go/bin:$PATH"
export DEBUG=true
dlv debug --listen=:2345 --headless=true --api-version=2 --accept-multiclient cmd/server/main.go
```

### Option 3: VS Code (Easiest - No Terminal Needed!)
1. Open VS Code
2. Press **F5**
3. Select "Launch Backend Server"
4. Done! Server starts in debug mode

## Attach VS Code to Running Debug Server

If you started the server with `start-debug.sh`:

1. Open VS Code Debug panel (`Cmd+Shift+D`)
2. Select **"Attach to Debug Server"** from dropdown
3. Press **F5**

## Permanent Fix for PATH

I've added `$HOME/go/bin` to your `~/.zshrc`. 

To apply immediately:
```bash
source ~/.zshrc
```

Or restart your terminal.

## Verify Delve Works

```bash
export PATH="$HOME/go/bin:$PATH"
dlv version
```

Should show: `Delve Debugger Version: 1.26.0`


