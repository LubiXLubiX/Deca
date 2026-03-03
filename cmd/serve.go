package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/spf13/cobra"
)

func compileJSLikeToESM(source string, loader api.Loader, absPath string) (string, error) {
	result := api.Transform(source, api.TransformOptions{
		Loader:            loader,
		Format:            api.FormatESModule,
		Target:            api.ES2020,
		Sourcemap:         api.SourceMapInline,
		Sourcefile:        absPath,
		JSX:               api.JSXTransform,
		JSXFactory:        "React.createElement",
		JSXFragment:       "React.Fragment",
		MinifyWhitespace:  false,
		MinifyIdentifiers: false,
		MinifySyntax:      false,
	})
	if len(result.Errors) > 0 {
		// Return first error for readability
		return "", fmt.Errorf(result.Errors[0].Text)
	}
	out := string(result.Code)
	// Classic JSX runtime expects `React` identifier in scope.
	// In a no-bundler environment, ensure it's always available.
	if loader == api.LoaderJSX || loader == api.LoaderTSX {
		// Inject React import if missing but used
		if (strings.Contains(out, "React.createElement") || strings.Contains(out, "React.Fragment")) && !strings.Contains(out, "import React") {
			out = "import React from 'https://esm.sh/react@18.2.0';\n" + out
		}
		// Map 'react' and 'react/jsx-runtime' to esm.sh URLs if they are used as imports
		out = strings.ReplaceAll(out, "from \"react\"", "from \"https://esm.sh/react@18.2.0\"")
		out = strings.ReplaceAll(out, "from 'react'", "from 'https://esm.sh/react@18.2.0'")
		out = strings.ReplaceAll(out, "from \"react/jsx-runtime\"", "from \"https://esm.sh/react@18.2.0/jsx-runtime\"")
		out = strings.ReplaceAll(out, "from 'react/jsx-runtime'", "from 'https://esm.sh/react@18.2.0/jsx-runtime'")
		out = strings.ReplaceAll(out, "from \"react-dom\"", "from \"https://esm.sh/react-dom@18.2.0\"")
		out = strings.ReplaceAll(out, "from 'react-dom'", "from 'https://esm.sh/react-dom@18.2.0'")
		out = strings.ReplaceAll(out, "from \"lucide-react\"", "from \"https://esm.sh/lucide-react\"")
		out = strings.ReplaceAll(out, "from 'lucide-react'", "from 'https://esm.sh/lucide-react'")
	}
	return out, nil
}

func buildBundledAppJS(frontendDir string) (string, error) {
	entryPath := filepath.Join(frontendDir, "src", "main.js")
	entrySource, err := os.ReadFile(entryPath)
	if err != nil {
		return "", err
	}

	// Build from stdin so we can force the loader to JSX even for .js entry files.
	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   string(entrySource),
			ResolveDir: filepath.Dir(entryPath),
			Sourcefile: entryPath,
			Loader:     api.LoaderJSX,
		},
		Bundle:            true,
		External:          []string{"https://*", "react", "react/jsx-runtime", "react/jsx-dev-runtime", "react-dom", "lucide-react"},
		Write:             false,
		Format:            api.FormatESModule,
		Platform:          api.PlatformBrowser,
		Target:            api.ES2020,
		Sourcemap:         api.SourceMapInline,
		JSX:               api.JSXTransform,
		JSXFactory:        "React.createElement",
		JSXFragment:       "React.Fragment",
		MinifyWhitespace:  false,
		MinifyIdentifiers: false,
		MinifySyntax:      false,
	})
	if len(result.Errors) > 0 {
		return "", fmt.Errorf(result.Errors[0].Text)
	}
	if len(result.OutputFiles) == 0 {
		return "", fmt.Errorf("no output from esbuild")
	}

	out := string(result.OutputFiles[0].Contents)
	// esbuild may emit bare imports for the JSX runtime. Browsers can't resolve these
	// without an import map. Rewrite them to esm.sh so /deca/app.js can run standalone.
	out = strings.ReplaceAll(out, "from \"react/jsx-runtime\"", "from \"https://esm.sh/react@18.2.0/jsx-runtime\"")
	out = strings.ReplaceAll(out, "from 'react/jsx-runtime'", "from 'https://esm.sh/react@18.2.0/jsx-runtime'")
	out = strings.ReplaceAll(out, "from \"react/jsx-dev-runtime\"", "from \"https://esm.sh/react@18.2.0/jsx-dev-runtime\"")
	out = strings.ReplaceAll(out, "from 'react/jsx-dev-runtime'", "from 'https://esm.sh/react@18.2.0/jsx-dev-runtime'")

	return out, nil
}

type sseHub struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

func newSSEHub() *sseHub {
	return &sseHub{clients: make(map[chan string]struct{})}
}

func (h *sseHub) add(c chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

func (h *sseHub) remove(c chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
}

func (h *sseHub) broadcast(msg string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c <- msg:
		default:
		}
	}
}

func findFrontendDir(projectRoot string) string {
	candidates := []string{
		filepath.Join(projectRoot, "resources", "app"),
		filepath.Join(projectRoot, "resources", "frontend"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			if _, err := os.Stat(filepath.Join(c, "index.html")); err == nil {
				return c
			}
		}
	}
	return ""
}

func startFrontendWatcher(frontendDir string, hub *sseHub, stop <-chan struct{}) {
	if frontendDir == "" {
		return
	}

	last := time.Now()
	_ = filepath.Walk(frontendDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.ModTime().After(last) {
			last = info.ModTime()
		}
		return nil
	})

	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()

	for {
		select {
		case <-stop:
			return
		case <-t.C:
			changed := false
			_ = filepath.Walk(frontendDir, func(p string, info os.FileInfo, err error) error {
				if err != nil || info == nil {
					return nil
				}
				if info.IsDir() {
					return nil
				}
				name := strings.ToLower(info.Name())
				if strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".jsx") || strings.HasSuffix(name, ".css") || strings.HasSuffix(name, ".html") {
					if info.ModTime().After(last) {
						last = info.ModTime()
						changed = true
					}
				}
				return nil
			})
			if changed {
				hub.broadcast("reload")
			}
		}
	}
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start Deca single-port development server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		projectRoot := detectProjectRoot(cwd)
		if projectRoot == "" {
			return fmt.Errorf("not in a LubiX project")
		}

		proxyPort := 3000

		fmt.Println("\n\033[1;34m--------------------------------------------------\033[0m")
		fmt.Printf("\033[1;32m[Deca] Unified Server: \033[1;36mhttp://localhost:%d\033[0m\n", proxyPort)
		fmt.Println("\033[1;34m--------------------------------------------------\033[0m")
		fmt.Println("\033[0;90mCleaning up ports...\033[0m")

		killPort(proxyPort)

		// PHP via FastCGI (2.b) - No internal port exposed
		phpCgiPath := "/opt/homebrew/bin/php-cgi"
		if _, err := os.Stat(phpCgiPath); err != nil {
			phpCgiPath = "php-cgi" // Fallback to PATH
		}

		frontendDir := findFrontendDir(projectRoot)
		hub := newSSEHub()
		watchStop := make(chan struct{})
		go startFrontendWatcher(frontendDir, hub, watchStop)

		mux := http.NewServeMux()

		// SSE live reload channel
		mux.HandleFunc("/__deca/live-reload", func(w http.ResponseWriter, r *http.Request) {
			fl, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "streaming unsupported", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			ch := make(chan string, 8)
			hub.add(ch)
			defer hub.remove(ch)

			fmt.Fprint(w, "event: ready\ndata: ok\n\n")
			fl.Flush()

			ctx := r.Context()
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-ch:
					fmt.Fprintf(w, "event: %s\ndata: 1\n\n", msg)
					fl.Flush()
				}
			}
		})

		// Bundled frontend entrypoint (single file) to avoid ESM/runtime interop issues
		bundleHandler := func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("\033[0;90m[Deca] Bundle %s %s\033[0m\n", r.Method, r.URL.Path)
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			if frontendDir == "" {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("throw new Error('frontend not found');"))
				return
			}
			js, err := buildBundledAppJS(frontendDir)
			if err != nil {
				fmt.Printf("\033[0;31m[Deca] Bundle error: %v\033[0m\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				safe := strings.ReplaceAll(err.Error(), "\\", "\\\\")
				safe = strings.ReplaceAll(safe, "\"", "\\\"")
				_, _ = w.Write([]byte("throw new Error(\"bundle error: " + safe + "\");"))
				return
			}
			_, _ = w.Write([]byte(js))
		}
		mux.HandleFunc("/__deca/app.js", bundleHandler)
		mux.HandleFunc("/deca/app.js", bundleHandler)

		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			p := r.URL.Path
			fmt.Printf("\033[0;90m[Deca] %s %s\033[0m\n", r.Method, p)

			// API route -> PHP via FastCGI (No port)
			if strings.HasPrefix(p, "/api/") {
				env := map[string]string{
					"SCRIPT_FILENAME": filepath.Join(projectRoot, "public", "index.php"),
					"SCRIPT_NAME":     "/index.php",
					"REQUEST_URI":     r.URL.RequestURI(),
					"DOCUMENT_ROOT":   filepath.Join(projectRoot, "public"),
					"GATEWAY_INTERFACE": "CGI/1.1",
					"SERVER_SOFTWARE":   "Deca/1.0",
					"REMOTE_ADDR":       r.RemoteAddr,
				}

				// Basic FastCGI implementation using php-cgi as a process
				cgiCmd := exec.Command(phpCgiPath)
				cgiCmd.Dir = filepath.Join(projectRoot, "public")
				
				// Map headers to CGI env
				for k, v := range r.Header {
					upperK := strings.ReplaceAll(strings.ToUpper(k), "-", "_")
					env["HTTP_"+upperK] = v[0]
				}
				env["REQUEST_METHOD"] = r.Method
				env["CONTENT_TYPE"] = r.Header.Get("Content-Type")
				env["CONTENT_LENGTH"] = r.Header.Get("Content-Length")
				
				cgiCmd.Env = os.Environ()
				for k, v := range env {
					cgiCmd.Env = append(cgiCmd.Env, k+"="+v)
				}

				stdin, _ := cgiCmd.StdinPipe()
				if r.Body != nil {
					go func() {
						defer stdin.Close()
						io.Copy(stdin, r.Body)
					}()
				}

				out, err := cgiCmd.CombinedOutput()
				if err != nil {
					http.Error(w, "PHP-CGI Error: "+err.Error()+"\n\n"+string(out), http.StatusInternalServerError)
					return
				}

				// Simple CGI response parser
				parts := strings.SplitN(string(out), "\r\n\r\n", 2)
				if len(parts) < 2 {
					w.Write(out)
					return
				}

				headerLines := strings.Split(parts[0], "\r\n")
				for _, line := range headerLines {
					headerParts := strings.SplitN(line, ": ", 2)
					if len(headerParts) == 2 {
						if strings.ToLower(headerParts[0]) == "status" {
							// Status: 200 OK
							statusParts := strings.SplitN(headerParts[1], " ", 2)
							if code, err := fmt.Sscanf(statusParts[0], "%d"); err == nil {
								w.WriteHeader(code)
							}
						} else {
							w.Header().Add(headerParts[0], headerParts[1])
						}
					}
				}
				w.Write([]byte(parts[1]))
				return
			}

			// Serve static files from public/ folder (images, favicons, etc.)
			publicDir := filepath.Join(projectRoot, "public")
			if strings.HasPrefix(p, "/assets/") || strings.HasSuffix(p, ".svg") || strings.HasSuffix(p, ".png") || strings.HasSuffix(p, ".jpg") || strings.HasSuffix(p, ".ico") {
				publicPath := filepath.Join(publicDir, filepath.Clean(p))
				if _, err := os.Stat(publicPath); err == nil {
					if strings.HasSuffix(p, ".svg") {
						w.Header().Set("Content-Type", "image/svg+xml")
					}
					http.ServeFile(w, r, publicPath)
					return
				}
			}

			// Frontend -> static from resources/app (fallback resources/frontend)
			if frontendDir == "" {
				http.Error(w, "frontend not found (expected resources/app/index.html)", http.StatusNotFound)
				return
			}

			// Serve index.html for SPA routes
			if p == "/" || !strings.Contains(filepath.Base(p), ".") {
				b, err := os.ReadFile(filepath.Join(frontendDir, "index.html"))
				if err != nil {
					http.Error(w, "failed to read index.html", http.StatusInternalServerError)
					return
				}
				html := string(b)
				// Inject live-reload script (dev only)
				inject := `<script>try{var es=new EventSource('/__deca/live-reload');es.addEventListener('reload',function(){location.reload();});}catch(e){}</script>`
				if strings.Contains(html, "</body>") {
					html = strings.Replace(html, "</body>", inject+"</body>", 1)
				} else {
					html += inject
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte(html))
				return
			}

			filePath := filepath.Join(frontendDir, filepath.Clean(p))

			// Handle JS/JSX/TS/TSX requests (compile on the fly to ESM)
			if strings.HasSuffix(p, ".js") || strings.HasSuffix(p, ".jsx") || strings.HasSuffix(p, ".ts") || strings.HasSuffix(p, ".tsx") {
				content, err := os.ReadFile(filePath)
				if err != nil {
					fmt.Printf("\033[0;31m[Deca] 404 File Not Found: %s\033[0m\n", filePath)
					http.NotFound(w, r)
					return
				}

				// Default to JSX loader for .js and .jsx to allow JSX syntax in both
				ext := strings.ToLower(filepath.Ext(p))
				loader := api.LoaderJSX
				if ext == ".ts" || ext == ".tsx" {
					loader = api.LoaderTSX
				}

				fmt.Printf("\033[0;90m[Deca] Transforming %s...\033[0m\n", p)
				compiled, cErr := compileJSLikeToESM(string(content), loader, filePath)
				if cErr != nil {
					fmt.Printf("\033[0;31m[Deca] Transform Error: %v\033[0m\n", cErr)
					http.Error(w, "JS compile error: "+cErr.Error(), http.StatusInternalServerError)
					return
				}
				fmt.Printf("\033[0;32m[Deca] Transform Success: %s\033[0m\n", p)
				w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
				_, _ = w.Write([]byte(compiled))
				return
			}

			// Prevent directory traversal
			if !strings.HasPrefix(filePath, frontendDir) {
				http.NotFound(w, r)
				return
			}
			http.ServeFile(w, r, filePath)
		})

		server := &http.Server{Addr: fmt.Sprintf(":%d", proxyPort), Handler: mux}
		go func() { _ = server.ListenAndServe() }()

		openBrowser(fmt.Sprintf("http://localhost:%d", proxyPort))

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		fmt.Println("\n\033[0;90mStopping Deca servers...\033[0m")
		close(watchStop)
		_ = server.Close()
		return nil
	},
}

func compileSvelteS2(source string, path string) string {
	_ = source
	_ = path
	return "throw new Error('Svelte support has been removed from this Deca build.');"
}
