package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Deca CLI to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("\n\033[1;34m[+] [Deca] Checking for updates...\033[0m")
		
		repoURL := "https://github.com/LubiXLubiX/Deca.git"
		tempDir := os.TempDir() + "/deca-upgrade"
		
		// 1. Clean old temp if exists
		os.RemoveAll(tempDir)

		// 2. Clone latest
		fmt.Println("\033[0;90m[+] Fetching latest source from GitHub...\033[0m")
		clone := exec.Command("git", "clone", "--depth", "1", repoURL, tempDir)
		if err := clone.Run(); err != nil {
			return fmt.Errorf("failed to fetch updates: %w", err)
		}

		// 3. Build new binary
		fmt.Println("\033[0;90m[+] Building new version...\033[0m")
		build := exec.Command("go", "build", "-o", "deca_new", "main.go")
		build.Dir = tempDir
		if err := build.Run(); err != nil {
			return fmt.Errorf("failed to build new version: %w", err)
		}

		// 4. Replace current binary
		fmt.Println("\033[0;90m[+] Installing...\033[0m")
		targetPath := "/usr/local/bin/deca"
		if runtime.GOOS == "windows" {
			targetPath = "C:\\Windows\\System32\\deca.exe"
		}

		// Use sudo for macOS/Linux replacement
		install := exec.Command("sudo", "mv", tempDir+"/deca_new", targetPath)
		if runtime.GOOS == "windows" {
			install = exec.Command("cmd", "/C", "move", "/Y", tempDir+"\\deca_new", targetPath)
		}

		if err := install.Run(); err != nil {
			return fmt.Errorf("failed to install new version (permission denied?): %w", err)
		}

		fmt.Println("\n\033[1;32m[OK] Deca has been upgraded successfully\033[0m")
		fmt.Println("Run 'deca version' to verify.")
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
