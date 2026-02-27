package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"
)

var runDevCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run LubiX dev environment (backend + vite)",
	RunE: func(cmd *cobra.Command, args []string) error {
		phpPort, _ := cmd.Flags().GetInt("php-port")
		vitePort, _ := cmd.Flags().GetInt("vite-port")

		cwd, _ := os.Getwd()
		projectRoot := detectProjectRoot(cwd)
		if projectRoot == "" {
			return fmt.Errorf("could not find LubiX project root (looking for public/ and package.json) in %s or its parents", cwd)
		}

		fmt.Println("\n\033[1;34mLubiX Professional Stack\033[0m")
		fmt.Println("--------------------------")
		fmt.Printf("\033[1;32m➜\033[0m  Local: \033[1;36mhttp://localhost:%d\033[0m\n", vitePort)
		fmt.Println("--------------------------")
		fmt.Println("\033[0;90mStarting servers...\033[0m\n")

		phpCmd := exec.Command("php", "-S", fmt.Sprintf("127.0.0.1:%d", phpPort), "-t", "public")
		phpCmd.Dir = projectRoot
		phpCmd.Stdout = os.Stdout
		phpCmd.Stderr = os.Stderr

		viteCmd := exec.Command("npm", "run", "dev", "--", "--port", fmt.Sprintf("%d", vitePort))
		viteCmd.Dir = projectRoot
		viteCmd.Stdout = os.Stdout
		viteCmd.Stderr = os.Stderr

		if err := phpCmd.Start(); err != nil {
			return fmt.Errorf("failed to start php server: %w", err)
		}
		if err := viteCmd.Start(); err != nil {
			_ = phpCmd.Process.Kill()
			return fmt.Errorf("failed to start vite: %w", err)
		}

		// Open browser (best-effort)
		_ = openBrowser(fmt.Sprintf("http://localhost:%d", vitePort))

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		done := make(chan error, 2)
		go func() { done <- phpCmd.Wait() }()
		go func() { done <- viteCmd.Wait() }()

		select {
		case <-sigCh:
			_ = terminateProcessTree(phpCmd)
			_ = terminateProcessTree(viteCmd)
			return nil
		case err := <-done:
			_ = terminateProcessTree(phpCmd)
			_ = terminateProcessTree(viteCmd)
			return err
		}
	},
}

func init() {
	runCmd.AddCommand(runDevCmd)
	runDevCmd.Flags().Int("php-port", 8000, "PHP backend port")
	runDevCmd.Flags().Int("vite-port", 5173, "Vite dev server port")
}

func detectProjectRoot(startDir string) string {
	curr := startDir
	for {
		if _, err := os.Stat(filepath.Join(curr, "public")); err == nil {
			if _, err := os.Stat(filepath.Join(curr, "package.json")); err == nil {
				return curr
			}
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return ""
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}

func terminateProcessTree(c *exec.Cmd) error {
	if c == nil || c.Process == nil {
		return nil
	}

	// Basic cross-platform termination.
	// For Windows we'd need more robust job objects; for now keep it simple.
	if runtime.GOOS == "windows" {
		return c.Process.Kill()
	}

	// Try graceful stop
	_ = c.Process.Signal(syscall.SIGTERM)
	return c.Process.Kill()
}

// Silence unused import errors on some platforms
var _ = runtime.GOMAXPROCS
