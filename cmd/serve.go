package cmd

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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
		phpHost := "127.0.0.1"

		fmt.Println("\n\033[1;34m--------------------------------------------------\033[0m")
		fmt.Printf("\033[1;32m🚀 [Deca] Unified Server: \033[1;36mhttp://localhost:%d\033[0m\n", proxyPort)
		fmt.Println("\033[1;34m--------------------------------------------------\033[0m")
		fmt.Println("\033[0;90mCleaning up ports...\033[0m")

		killPort(proxyPort)

		// Start PHP on a random localhost port (not exposed). Still single public port: 3000.
		ln, err := net.Listen("tcp", phpHost+":0")
		if err != nil {
			return fmt.Errorf("failed to allocate php port: %w", err)
		}
		phpPort := ln.Addr().(*net.TCPAddr).Port
		_ = ln.Close()

		phpCmd := exec.Command("php", "-S", fmt.Sprintf("%s:%d", phpHost, phpPort), "-t", "public")
		phpCmd.Dir = projectRoot
		phpCmd.Stdout = io.Discard
		phpCmd.Stderr = io.Discard
		if err := phpCmd.Start(); err != nil {
			return fmt.Errorf("failed to start php: %w", err)
		}

		targetBackend, _ := url.Parse(fmt.Sprintf("http://%s:%d", phpHost, phpPort))
		proxyBackend := httputil.NewSingleHostReverseProxy(targetBackend)

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

			// API route -> PHP backend (internal only)
			if strings.HasPrefix(p, "/api/") {
				proxyBackend.ServeHTTP(w, r)
				return
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
		_ = phpCmd.Process.Kill()
		_ = server.Close()
		return nil
	},
}
