# Troubleshooting MIME Type Error

## Error: "Expected a JavaScript-or-Wasm module script but the server responded with a MIME type of 'application/octet-stream'"

This error occurs when:
1. **Opening HTML file directly** - Don't open `index.html` directly in the browser
2. **Vite dev server not running** - The dev server must be running
3. **Wrong server** - Using a static file server instead of Vite

## Solution

### ✅ Correct Way to Start

1. **Make sure you're in the frontend directory:**
   ```bash
   cd frontend
   ```

2. **Install dependencies (first time only):**
   ```bash
   npm install
   ```

3. **Start the Vite dev server:**
   ```bash
   npm run dev
   ```

4. **Access the app through the dev server URL:**
   - ✅ **Correct**: http://localhost:3000
   - ❌ **Wrong**: file:///path/to/index.html

### ❌ Common Mistakes

1. **Double-clicking index.html** - This opens it as a file, not through a server
2. **Using a simple HTTP server** - Python's `http.server` or similar won't work with Vite
3. **Opening in browser without server** - The browser can't process TypeScript/JSX without Vite

### Verify Vite is Running

When you run `npm run dev`, you should see:
```
  VITE v5.x.x  ready in xxx ms

  ➜  Local:   http://localhost:3000/
  ➜  Network: use --host to expose
```

### If Still Having Issues

1. **Clear browser cache:**
   - Hard refresh: `Cmd+Shift+R` (Mac) or `Ctrl+Shift+R` (Windows/Linux)

2. **Check port 3000 is available:**
   ```bash
   lsof -i :3000
   ```

3. **Try a different port:**
   ```bash
   npm run dev -- --port 3001
   ```

4. **Delete node_modules and reinstall:**
   ```bash
   rm -rf node_modules package-lock.json
   npm install
   npm run dev
   ```

5. **Check Vite version:**
   ```bash
   npx vite --version
   ```

## Why This Happens

Vite uses ES modules and needs to:
- Transform TypeScript/JSX to JavaScript
- Handle module imports correctly
- Set proper MIME types for JavaScript modules
- Provide hot module replacement (HMR)

A static file server can't do this - only Vite's dev server can!


