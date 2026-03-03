<div align="center">
<img src="/decabanner.png" alt="Deca Logo" width="100%">

<h1>🚀 Deca CLI</h1>
<p><b>The ultimate CLI for the LubiX Framework. Zero-config, single-port, full-stack React + PHP development.</b></p>

</div>

💡 The Main Goal

Deca is a blazing-fast, cross-platform command-line interface designed to power the LubiX ecosystem. It bridges the gap between modern React reactivity and stable PHP backends by providing a unified developer experience without the usual configuration headaches.

Why use Deca?

🔌 One Port for Everything: Serve both your frontend React assets and proxy your PHP API requests seamlessly through a single port (3000). No CORS issues, no multiple terminal tabs.

⚡ Zero Node/Vite Required: Write React/JSX and see changes instantly. No npm install, no heavy node_modules, and no complex Vite/Webpack build steps for local development.

🐘 FastCGI Integration: Communicates directly with php-cgi. Your PHP backend is processed internally without exposing additional internal ports.

📦 On-the-fly Bundling: Powered by esbuild under the hood, compiling your JSX/TSX instantly.

🔄 Live Reload (SSE): Automatic, lightning-fast browser refresh when you save your code using Server-Sent Events.

⚙️ Requirements

Before installing Deca, ensure your system has the following:

PHP (8.0+ recommended) with php-cgi available in your system PATH.

Composer (for managing PHP dependencies).

Note: Even if you are only running the frontend, Deca still requires php-cgi because the unified server is fundamentally designed to handle both environments simultaneously.

📥 Installation

Option A: Pre-compiled Binaries (Recommended)

Download the latest pre-compiled binary for your operating system from the GitHub Releases page.

<details>
<summary><b>🍎 macOS (Intel & Apple Silicon)</b></summary>

Download deca_darwin_arm64 (for M1/M2/M3) or deca_darwin_amd64 (for Intel).

Rename the downloaded file to deca.

Make it executable and move it to your PATH:

chmod +x deca
sudo mv deca /usr/local/bin/deca


Verify installation:

deca version


</details>

<details>
<summary><b>🐧 Linux</b></summary>

Download deca_linux_amd64.

Rename the file to deca.

Make it executable and move it to your PATH:

chmod +x deca
sudo mv deca /usr/local/bin/deca


Verify installation:

deca version


</details>

<details>
<summary><b>🪟 Windows (PowerShell)</b></summary>

Download deca_windows_amd64.exe.

Rename the file to deca.exe.

Create a bin folder in your user directory:

mkdir $env:USERPROFILE\bin


Move the executable:

move .\deca.exe $env:USERPROFILE\bin\deca.exe


Add it to your PATH:

Search Windows for "Edit environment variables for your account".

Select Path and click Edit -> New.

Add %USERPROFILE%\bin.

Restart your terminal.

Verify installation:

deca version


</details>

Option B: Build from Source

If you have Go 1.21+ installed, you can compile Deca yourself:

git clone [https://github.com/LubiXLubiX/Deca.git](https://github.com/LubiXLubiX/Deca.git)
cd Deca/Deca-CLI
go build -o deca main.go

# Move the resulting 'deca' binary to your PATH based on your OS.


🚀 Quickstart

Get a full-stack React + PHP app running in seconds:

# 1. Scaffold a new LubiX project
deca create-project my-app

# 2. Enter the directory
cd my-app

# 3. Start the unified development server
deca lubix serve


What happens under the hood? > deca create-project automatically copies .env.example to .env and attempts to run composer install. If it fails (due to missing PHP extensions), you can run composer install manually.

🛠️ Command Reference

Deca comes with a powerful set of commands to manage your entire development lifecycle.

🟢 Core & Server

Command

Description

deca version

Show the currently installed version of Deca.

deca doctor

Check system requirements (PHP, CGI, Composer, etc.) and diagnose issues.

deca create-project <name>

Scaffold a fresh LubiX React+PHP project.

deca lubix serve

Start the unified development server at http://localhost:3000.

deca lubix dev

Alias for deca lubix serve.

🔵 Database & Migrations

Command

Description

deca lubix db:create

Create the database defined in your .env file.

deca lubix migrate

Execute all pending database migrations.

deca lubix migrate:rollback [N]

Rollback the last N batches of migrations (default is 1).

🟣 Generators (Scaffolding)

Command

Description

deca lubix make:controller <Name>

Generate a new PHP Controller class.

deca lubix make:model <Name>

Generate a new PHP Model class.

deca lubix make:migration <Name>

Generate a new database migration file.

🔄 Upgrading Deca CLI

Deca can update itself seamlessly.

Automatic Upgrade:

deca upgrade


This command fetches the latest source from GitHub, compiles a new binary, and replaces your current deca executable. (If you get a permission denied error, run it with sudo on Unix systems).

Manual Upgrade:
If you prefer manual control:

git clone [https://github.com/LubiXLubiX/Deca.git](https://github.com/LubiXLubiX/Deca.git)
cd Deca/Deca-CLI
git pull origin main
go build -o deca main.go
# Replace your current binary with the newly built one


🐛 Troubleshooting

❌ "php-cgi" not found

Deca requires php-cgi for backend processing to work.

macOS: brew install php

Linux (Ubuntu/Debian): sudo apt install php-cgi

Windows: Ensure the PHP folder containing php-cgi.exe is in your system PATH.

Run deca doctor to verify your environment setup.

❌ Permission Denied (macOS/Linux)

If you cannot execute the deca command, ensure the binary has execution rights:

chmod +x /path/to/deca


❌ Blank Screen on http://localhost:3000

If the browser shows a white screen:

Open the Browser Console (F12) to check for React syntax errors.

Check the Network Tab and ensure /deca/app.js returns a 200 OK status.

How it works: Deca rewrites React JSX runtime imports (e.g., react/jsx-runtime) to point to https://esm.sh/.... Ensure you have an active internet connection so the browser can fetch these ES modules.

📄 License

Deca CLI and the LubiX framework are distributed under the MIT License. See LICENSE for more information.

<div align="center">
<i>Built with ❤️ by Rakhan for the modern web.</i>
</div>
