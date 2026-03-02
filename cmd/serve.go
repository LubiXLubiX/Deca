package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

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
		phpPort := 8000
		vitePort := 5173

		fmt.Println("\n\033[1;34m--------------------------------------------------\033[0m")
		fmt.Printf("\033[1;32m🚀 [Deca] Unified Server: \033[1;36mhttp://localhost:%d\033[0m\n", proxyPort)
		fmt.Println("\033[1;34m--------------------------------------------------\033[0m")
		fmt.Println("\033[0;90mCleaning up ports...\033[0m")

		killPort(proxyPort)
		killPort(phpPort)
		killPort(vitePort)

		// 1. Start PHP (Silent)
		phpCmd := exec.Command("php", "-S", fmt.Sprintf("127.0.0.1:%d", phpPort), "-t", "public")
		phpCmd.Dir = projectRoot
		phpCmd.Stdout = io.Discard
		phpCmd.Stderr = io.Discard
		if err := phpCmd.Start(); err != nil {
			return fmt.Errorf("failed to start php: %w", err)
		}

		// 2. Start Vite (Silent)
		viteCmd := exec.Command("npm", "run", "dev", "--", "--port", fmt.Sprintf("%d", vitePort))
		viteCmd.Dir = projectRoot
		viteCmd.Stdout = io.Discard
		viteCmd.Stderr = io.Discard
		if err := viteCmd.Start(); err != nil {
			phpCmd.Process.Kill()
			return fmt.Errorf("failed to start vite: %w", err)
		}

		fmt.Println("\033[0;90mServers are running in background...\033[0m")

		// 3. Proxy Logic
		targetBackend, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", phpPort))
		targetFrontend, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", vitePort))

		proxyBackend := httputil.NewSingleHostReverseProxy(targetBackend)
		proxyFrontend := httputil.NewSingleHostReverseProxy(targetFrontend)

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/src") || 
			   strings.HasPrefix(r.URL.Path, "/@vite") || 
			   strings.HasPrefix(r.URL.Path, "/@fs") ||
			   strings.HasPrefix(r.URL.Path, "/node_modules") || 
			   r.Header.Get("Upgrade") == "websocket" {
				proxyFrontend.ServeHTTP(w, r)
			} else {
				proxyBackend.ServeHTTP(w, r)
			}
		})

		server := &http.Server{Addr: fmt.Sprintf(":%d", proxyPort)}
		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("Proxy error: %v\n", err)
			}
		}()

		openBrowser(fmt.Sprintf("http://localhost:%d", proxyPort))

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		fmt.Println("\n\033[0;90mStopping Deca servers...\033[0m")
		phpCmd.Process.Kill()
		viteCmd.Process.Kill()
		server.Close()
		return nil
	},
}

func detectProjectRoot(startDir string) string {
	curr := startDir
	for {
		if _, err := os.Stat(filepath.Join(curr, "public")); err == nil {
			return curr
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return ""
}

func killPort(port int) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", fmt.Sprintf("for /f \"tokens=5\" %%a in ('netstat -aon ^| findstr :%d ^| findstr LISTENING') do taskkill /F /PID %%a", port))
	} else {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("lsof -ti:%d | xargs kill -9 2>/dev/null", port))
	}
	_ = cmd.Run()
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}
