package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		currentExe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to locate current deca executable: %w", err)
		}
		currentExe, _ = filepath.EvalSymlinks(currentExe)
		
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
		newExe := "deca_new"
		if runtime.GOOS == "windows" {
			newExe = "deca_new.exe"
		}
		build := exec.Command("go", "build", "-o", newExe, "main.go")
		build.Dir = tempDir
		if err := build.Run(); err != nil {
			return fmt.Errorf("failed to build new version: %w", err)
		}

		// 4. Replace current binary
		fmt.Println("\033[0;90m[+] Installing...\033[0m")
		src := filepath.Join(tempDir, newExe)
		if runtime.GOOS == "windows" {
			install := exec.Command("cmd", "/C", "move", "/Y", src, currentExe)
			if err := install.Run(); err != nil {
				return fmt.Errorf("failed to install new version: %w", err)
			}
		} else {
			// Try without sudo first (works if user installed to a writable location)
			if err := os.Rename(src, currentExe); err != nil {
				// If permission denied, attempt via sudo (common for /usr/local/bin)
				install := exec.Command("sudo", "mv", src, currentExe)
				if sErr := install.Run(); sErr != nil {
					fmt.Fprintln(os.Stderr, "[!] Failed to replace the current deca binary.")
					fmt.Fprintln(os.Stderr, "[!] Manual upgrade:")
					fmt.Fprintf(os.Stderr, "    git clone %s\n", repoURL)
					fmt.Fprintln(os.Stderr, "    cd Deca-CLI")
					fmt.Fprintln(os.Stderr, "    go build -o deca")
					fmt.Fprintf(os.Stderr, "    sudo mv deca %s\n", currentExe)
					return fmt.Errorf("failed to install new version: %w", sErr)
				}
			}
		}

		fmt.Println("\n\033[1;32m[OK] Deca has been upgraded successfully\033[0m")
		fmt.Println("Run 'deca version' to verify.")
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
