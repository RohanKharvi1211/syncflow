# VS Code Debugging Guide

## Prerequisites

1. **Install Go Extension** in VS Code:
   - Open VS Code
   - Go to Extensions (Cmd+Shift+X)
   - Search for "Go" by Google
   - Install it

2. **Install Delve** (Go debugger):
   ```bash
   go install github.com/go-delve/delve/cmd/dlv@latest
   ```

## How to Debug

### Method 1: Using VS Code Debug Panel (Recommended)

1. **Open the Debug Panel:**
   - Press `Cmd+Shift+D` (Mac) or `Ctrl+Shift+D` (Windows/Linux)
   - Or click the Debug icon in the sidebar

2. **Select Configuration:**
   - At the top, select "Launch Backend Server" from the dropdown

3. **Set Breakpoints:**
   - Click in the left margin (gutter) next to any line number
   - A red dot will appear indicating a breakpoint

4. **Start Debugging:**
   - Press `F5` or click the green play button
   - The server will start and pause at your breakpoints

5. **Debug Controls:**
   - **Continue** (F5): Resume execution
   - **Step Over** (F10): Execute current line
   - **Step Into** (F11): Step into function calls
   - **Step Out** (Shift+F11): Step out of current function
   - **Restart** (Cmd+Shift+F5): Restart debugging
   - **Stop** (Shift+F5): Stop debugging

### Method 2: Using Command Palette

1. Press `Cmd+Shift+P` (Mac) or `Ctrl+Shift+P` (Windows/Linux)
2. Type "Debug: Start Debugging"
3. Select "Launch Backend Server"

## Debug Features

### Variables Panel
- View all variables in the current scope
- Hover over variables to see their values
- Right-click to "Set Value" for testing

### Watch Panel
- Add expressions to watch
- Monitor variable values as you step through code

### Call Stack
- See the function call hierarchy
- Navigate to different stack frames

### Debug Console
- Evaluate expressions
- Print variable values
- Execute Go code in the current context

## Example: Debugging User Handler

1. Open `backend/internal/controllers/user_handler.go`
2. Set a breakpoint on line 28 (inside `GetUsers` function)
3. Start debugging (F5)
4. Make a request to `http://localhost:8080/api/users?email=test@example.com`
5. Execution will pause at your breakpoint
6. Inspect variables, step through code, etc.

## Environment Variables

The debug configuration sets `DEBUG=true` automatically. You can also set other env vars in `launch.json`:

```json
"env": {
  "DEBUG": "true",
  "DB_HOST": "localhost",
  "DB_USER": "postgres",
  "DB_PASSWORD": "your_password",
  "DB_NAME": "syncflow",
  "DB_PORT": "5432",
  "PORT": "8080"
}
```

## Troubleshooting

### "Cannot find package" errors
- Run `go mod tidy` in the backend directory
- Make sure `GOPATH` and `GOROOT` are set correctly

### Debugger won't start
- Check that Delve is installed: `dlv version`
- Restart VS Code
- Check the Debug Console for error messages

### Breakpoints not working
- Make sure you're using the correct launch configuration
- Check that the file path matches
- Try setting breakpoints on executable lines (not comments/blank lines)

## Quick Debug Tips

1. **Conditional Breakpoints:**
   - Right-click on a breakpoint
   - Add a condition (e.g., `user.Email == "test@example.com"`)

2. **Logpoints:**
   - Right-click in the gutter
   - Select "Add Logpoint"
   - Add a message like: `User ID: {user.ID}`

3. **Debug Console Commands:**
   - `print variableName` - Print variable value
   - `locals` - Show all local variables
   - `args` - Show function arguments


