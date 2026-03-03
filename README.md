# Deca CLI
![Deca Logo](/decabanner.png)

Deca is a cross-platform command-line interface designed for a **zero-config, single-port** development experience in the LubiX ecosystem.

## The Main Goal
Deca bridges the gap between modern React reactivity and stable PHP backends by providing a **unified developer experience**.
- **One Port (`3000`)** for everything (frontend + backend).
- **Zero Node/Vite** requirement for local development.
- **FastCGI Integration** for PHP (No internal ports exposed).
- **On-the-fly React Bundling** with esbuild.

---

## Features
- **Unified Dev Server**: Serves your frontend assets and proxies API requests seamlessly.
- **Zero-Build React**: Write React/JSX and see changes instantly without `npm install` or complex build steps.
- **Live Reload (SSE)**: Automatic browser refresh when you save your code.
- **Database Management**: Integrated commands for migrations and rollbacks.
- **Code Generation**: Generate controllers, models, and migrations.
- **Cross-Platform**: Full support for macOS (Intel/Apple Silicon), Linux, and Windows.

---

## Installation

### Requirements
- **PHP** with `php-cgi` available in PATH.
- **Composer**.

If you only run the frontend, Deca still needs `php-cgi` because the unified server is designed to serve frontend + backend from one port.

### Option A: From GitHub Releases (Recommended)
Download the pre-compiled binary for your system from the [Releases](https://github.com/LubiXLubiX/Deca/releases) page.

#### macOS (Intel / Apple Silicon)
1. Download `deca_darwin_arm64` (Apple Silicon) or `deca_darwin_amd64` (Intel).
2. Rename the file to `deca`.
3. Move it to your PATH:
   ```bash
   chmod +x deca
   sudo mv deca /usr/local/bin/deca
   ```
4. Verify: `deca version`

#### Linux
1. Download `deca_linux_amd64`.
2. Rename the file to `deca`.
3. Move it to your PATH:
   ```bash
   chmod +x deca
   sudo mv deca /usr/local/bin/deca
   ```
4. Verify: `deca version`

#### Windows (PowerShell)
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
go build -o deca main.go
# Move the 'deca' binary to your PATH as shown above.
```

---

## Quickstart

Create a new LubiX project and start the unified server:
```bash
deca create-project my-app
cd my-app
deca lubix serve
```

Notes:
- `deca create-project` automatically copies `.env.example` to `.env` and attempts to run `composer install`.
- If Composer fails (network / PHP extensions), run `composer install` manually inside the project.

---

## Command Reference

| Command | Description |
| :--- | :--- |
| `deca version` | Show current version |
| `deca upgrade` | Upgrade Deca CLI to the latest version (auto) |
| `deca create-project <name>` | Create a new LubiX project |
| `deca lubix serve` | Start unified server at `http://localhost:3000` |
| `deca lubix dev` | Alias for `deca lubix serve` |
| `deca lubix migrate` | Execute pending database migrations |
| `deca lubix migrate:rollback [N]` | Rollback N batches of migrations |
| `deca lubix db:create` | Create the database defined in .env |
| `deca lubix make:controller <Name>` | Generate a controller |
| `deca lubix make:model <Name>` | Generate a model |
| `deca lubix make:migration <name>` | Generate a migration |
| `deca doctor` | Check system requirements (PHP, CGI, etc.) |

---

## Upgrade Deca CLI

### Automatic Upgrade
```bash
deca upgrade
```
This will:
- Fetch the latest source from GitHub
- Build a new binary
- Replace your current `deca` binary
- If permission denied, it will show manual steps

### Manual Upgrade
If automatic upgrade fails or you prefer manual control:
```bash
git clone https://github.com/LubiXLubiX/Deca.git
cd Deca/Deca-CLI
git pull
go build -o deca main.go
# Replace your current binary with the new one
sudo mv deca /usr/local/bin/deca  # macOS/Linux
# or
move .\deca.exe $env:USERPROFILE\bin\deca.exe  # Windows
```

---

## Troubleshooting

### "php-cgi" not found
Deca requires `php-cgi` for backend processing.
- **macOS**: `brew install php`
- **Linux**: `sudo apt install php-cgi` (or equivalent)
- **Verify**: Run `deca doctor` to check your environment.

### "Permission Denied" (macOS/Linux)
Ensure the binary is executable: `chmod +x /path/to/deca`

### Blank Screen on http://localhost:3000
If the page is blank:
- Check the browser Console.
- Check the Network tab and confirm `/deca/app.js` returns 200.

Deca rewrites React JSX runtime imports (for example `react/jsx-runtime`) to `https://esm.sh/...` so the browser can load the bundle without Node/Vite.

---

## License
Distributed under the MIT License. See `LICENSE` for more information.
