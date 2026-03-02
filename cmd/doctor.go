package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system health",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("\033[1;34m🏥 Deca Doctor - System Check\033[0m")
		deps := []string{"php", "node", "composer", "npm"}
		for _, dep := range deps {
			if _, err := exec.LookPath(dep); err != nil {
				fmt.Printf("✘ %s: Not found\n", dep)
			} else {
				fmt.Printf("✔ %s: Installed\n", dep)
			}
		}
		fmt.Println("OS:", runtime.GOOS)
	},
}
