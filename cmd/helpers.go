package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func detectProjectRoot(startDir string) string {
	curr := startDir
	for {
		// Check for common LubiX project markers
		markers := []string{"public", "composer.json", "packages/lubix-cli"}
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(curr, marker)); err == nil {
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
	if cmd != nil {
		_ = cmd.Start()
	}
}
