package cmd

import (
	"fmt"
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

		fmt.Printf("\033[1;34m🚀 [Deca] Starting Unified Server on http://localhost:%d\033[0m\n", proxyPort)

		killPort(proxyPort)
		killPort(phpPort)
		killPort(vitePort)

		// 1. Start PHP
		phpCmd := exec.Command("php", "-S", fmt.Sprintf("127.0.0.1:%d", phpPort), "-t", "public")
		phpCmd.Dir = projectRoot
		phpCmd.Stdout = os.Stdout
		phpCmd.Stderr = os.Stderr
		phpCmd.Start()

		// 2. Start Vite
		viteCmd := exec.Command("npm", "run", "dev", "--", "--port", fmt.Sprintf("%d", vitePort))
		viteCmd.Dir = projectRoot
		viteCmd.Stdout = os.Stdout
		viteCmd.Stderr = os.Stderr
		viteCmd.Start()

		// 3. Proxy Logic
		targetBackend, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", phpPort))
		targetFrontend, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", vitePort))

		proxyBackend := httputil.NewSingleHostReverseProxy(targetBackend)
		proxyFrontend := httputil.NewSingleHostReverseProxy(targetFrontend)

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Proxy /assets or HMR or Vite paths to Frontend, else Backend
			if strings.HasPrefix(r.URL.Path, "/src") || strings.HasPrefix(r.URL.Path, "/@vite") || strings.HasPrefix(r.URL.Path, "/node_modules") || r.Header.Get("Upgrade") == "websocket" {
				proxyFrontend.ServeHTTP(w, r)
			} else {
				proxyBackend.ServeHTTP(w, r)
			}
		})

		server := &http.Server{Addr: fmt.Sprintf(":%d", proxyPort)}
		go server.ListenAndServe()

		openBrowser(fmt.Sprintf("http://localhost:%d", proxyPort))

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		fmt.Println("\nStopping Deca servers...")
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
