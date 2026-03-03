# Deca CLI

Deca is a cross-platform command-line interface designed for a **zero-config, single-port** development experience in the LubiX ecosystem.

## 🚀 The Main Goal
Deca bridges the gap between modern React reactivity and stable PHP backends by providing a **unified developer experience**.
- **One Port (`3000`)** for everything (frontend + backend).
- **Zero Node/Vite** requirement for local development.
- **FastCGI Integration** for PHP (No internal ports exposed).
- **On-the-fly React Bundling** with esbuild.

---

## 🛠️ Features
- **Unified Dev Server**: Serves your frontend assets and proxies API requests seamlessly.
- **Zero-Build React**: Write React/JSX and see changes instantly without `npm install` or complex build steps.
- **Live Reload (SSE)**: Automatic browser refresh when you save your code.
- **Database Management**: Integrated commands for migrations and rollbacks.
- **Cross-Platform**: Full support for macOS, Linux, and Windows.

---

## 📦 Installation

### Option A: From GitHub Releases (Recommended)
Download the pre-compiled binary for your system from the [Releases](https://github.com/LubiXLubiX/Deca/releases) page.

#### 🍎 macOS (Intel / Apple Silicon)
1. Download `deca_darwin_arm64` (Apple Silicon) or `deca_darwin_amd64` (Intel).
2. Rename the file to `deca`.
3. Move it to your PATH:
   ```bash
   chmod +x deca
   sudo mv deca /usr/local/bin/deca
   ```
4. Verify: `deca version`

#### 🐧 Linux
1. Download `deca_linux_amd64`.
2. Rename the file to `deca`.
3. Move it to your PATH:
   ```bash
   chmod +x deca
   sudo mv deca /usr/local/bin/deca
   ```
4. Verify: `deca version`

#### 🪟 Windows (PowerShell)
1. Download `deca_windows_amd64.exe`.
2. Rename the file to `deca.exe`.
3. Create a folder for binaries: `mkdir $env:USERPROFILE\bin`
4. Move the file: `move .\deca.exe $env:USERPROFILE\bin\deca.exe`
5. Add to PATH: Search for "Edit environment variables" -> Path -> Add `%USERPROFILE%\bin`
6. Verify: `deca version`

### Option B: Build from Source
If you have **Go 1.21+** installed:
```bash
git clone https://github.com/LubiXLubiX/Deca.git
cd Deca/Deca-CLI
go build -o deca
# Move the 'deca' binary to your PATH as shown above.
```

---

## ⚡ Quickstart
From your LubiX project root:
```bash
# Start the unified dev server
deca lubix serve

# Run database migrations
deca lubix migrate

# Rollback migrations (1 batch)
deca lubix migrate:rollback
```

---

## 📜 Command Reference

| Command | Description |
| :--- | :--- |
| `deca version` | Show current version |
| `deca lubix serve` | Start unified server at `http://localhost:3000` |
| `deca lubix migrate` | Execute pending database migrations |
| `deca lubix migrate:rollback [N]` | Rollback N batches of migrations |
| `deca doctor` | Check system requirements (PHP, CGI, etc.) |

---

## 🔧 Troubleshooting

### "php-cgi" not found
Deca requires `php-cgi` for backend processing.
- **macOS**: `brew install php`
- **Linux**: `sudo apt install php-cgi` (or equivalent)
- **Verify**: Run `deca doctor` to check your environment.

### "Permission Denied" (macOS/Linux)
Ensure the binary is executable: `chmod +x /path/to/deca`

### Blank Screen on http://localhost:3000
1. Open Browser DevTools (F12).
2. Check the **Console** for errors.
3. Check the **Network** tab to ensure `/deca/app.js` is loading (Status 200).

---

## 🛡️ License
Distributed under the MIT License. See `LICENSE` for more information.
