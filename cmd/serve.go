package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

type sseHub struct {
	clients map[chan string]struct{}
}

func newSSEHub() *sseHub {
	return &sseHub{clients: make(map[chan string]struct{})}
}

func (h *sseHub) add(c chan string) {
	h.clients[c] = struct{}{}
}

func (h *sseHub) remove(c chan string) {
	delete(h.clients, c)
}

func (h *sseHub) broadcast(msg string) {
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
				if strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".css") || strings.HasSuffix(name, ".html") || strings.HasSuffix(name, ".svelte") {
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
		fmt.Printf("\033[1;32m🚀 [Deca] Unified Server: \033[1;36mhttp://localhost:%d\033[0m\n", proxyPort)
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

			// Handle .js imports that might actually be .svelte files (Vite style)
			if strings.HasSuffix(p, ".js") {
				// Check if a .svelte file exists with the same name
				sveltePath := strings.TrimSuffix(filePath, ".js") + ".svelte"
				if _, err := os.Stat(sveltePath); err == nil {
					content, _ := os.ReadFile(sveltePath)
					js := compileSvelteS2(string(content), p)
					w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
					_, _ = w.Write([]byte(js))
					return
				}
			}

			// Svelte S2: On-the-fly compilation for .svelte files
			if strings.HasSuffix(p, ".svelte") {
				content, err := os.ReadFile(filePath)
				if err != nil {
					fmt.Printf("\033[0;31m[Deca] 404 Svelte: %s\033[0m\n", filePath)
					http.NotFound(w, r)
					return
				}
				// Minimalistic Svelte S2 Compiler (Regex-based transform for dev)
				js := compileSvelteS2(string(content), p)
				w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
				_, _ = w.Write([]byte(js))
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
	componentName := filepath.Base(path)
	componentName = strings.TrimSuffix(componentName, ".svelte")
	componentName = strings.TrimSuffix(componentName, ".js")

	// Extract script, template, and style
	scriptRegex := regexp.MustCompile(`(?s)<script>(.*?)</script>`)
	styleRegex := regexp.MustCompile(`(?s)<style>(.*?)</style>`)

	scriptMatch := scriptRegex.FindStringSubmatch(source)
	script := ""
	if len(scriptMatch) > 1 {
		script = scriptMatch[1]
	}

	styleMatch := styleRegex.FindStringSubmatch(source)
	style := ""
	if len(styleMatch) > 1 {
		style = styleMatch[1]
	}

	template := scriptRegex.ReplaceAllString(source, "")
	template = styleRegex.ReplaceAllString(template, "")
	template = strings.TrimSpace(template)
	
	// Escape template for safe JS string embedding
	escapeForJS := func(s string) string {
		s = strings.ReplaceAll(s, "\\", "\\\\")
		s = strings.ReplaceAll(s, "\"", "\\\"")
		s = strings.ReplaceAll(s, "`", "\\x60")
		s = strings.ReplaceAll(s, "${", "\\${")
		s = strings.ReplaceAll(s, "\n", "\\n")
		s = strings.ReplaceAll(s, "\r", "")
		return s
	}
	
	escapedTemplate := escapeForJS(template)
	escapedStyle := escapeForJS(style)

	// Transform script: change `let x = y` to `this.x = y` for reactive variables
	// and wrap functions to maintain this context
	letRegex := regexp.MustCompile(`\blet\s+(\w+)\s*=`)
	script = letRegex.ReplaceAllString(script, "this.$1 =")
	constRegex := regexp.MustCompile(`\bconst\s+(\w+)\s*=\s*\(`)
	script = constRegex.ReplaceAllString(script, "this.$1 = (")
	
	// Build the JS - proper class with execution context
	js := fmt.Sprintf(`/* Deca S2: %s */
export default class %s {
    constructor(opts) {
        this.target = opts.target;
        this.props = opts.props || {};
        this.count = 0;
        this.name = 'User';
        Object.assign(this, this.props);
        this.init();
    }
    init() {
        try {
            // Execute script in component context
            (function(){
                %s
            }).call(this);
        } catch(e) { 
            console.error("[S2] Init error:", e); 
        }
        this.render();
    }
    render() {
        try {
            console.log('[S2] Rendering...', this.constructor.name);
            let html = "%s";
            // Interpolate ${var}
            html = html.replace(/\$\{([^}]+)\}/g, (match, expr) => {
                try { 
                    const val = (new Function('return ' + expr)).call(this);
                    return val !== undefined ? val : ''; 
                } catch(e) { 
                    return ''; 
                }
            });
            this.target.innerHTML = html;
            
            // Bind events
            this.target.querySelectorAll('[on\\:click]').forEach(el => {
                const code = el.getAttribute('on:click');
                el.addEventListener('click', () => {
                    (new Function('return ' + code)).call(this);
                });
            });
            
            // Inject styles
            const css = "%s";
            if (css) {
                let s = document.getElementById('s2-'+this.constructor.name);
                if (!s) {
                    s = document.createElement('style');
                    s.id = 's2-'+this.constructor.name;
                    document.head.appendChild(s);
                }
                s.textContent = css.replace(/\$\{[^}]+\}/g, '');
                console.log('[S2] CSS injected');
            }
            console.log('[S2] Render complete');
        } catch(e) {
            console.error("[S2] Render error:", e);
            this.target.innerHTML = '<div style="color:red;padding:20px">S2 Error: '+e.message+'</div>';
        }
    }
}`, path, componentName, script, escapedTemplate, escapedStyle)

	return js
}
