package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLubixCmd("migrate")
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "migrate:rollback",
	Short: "Rollback migrations",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		command := "migrate:rollback"
		if len(args) == 1 {
			command = fmt.Sprintf("migrate:rollback . %s", args[0])
		}
		return runLubixCmd(command)
	},
}

func runLubixCmd(command string) error {
	cwd, _ := os.Getwd()
	root := detectProjectRoot(cwd)
	c := exec.Command("php", "lubix", command)
	c.Dir = root
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	fmt.Printf("[+] [Deca] Running deca lubix %s...\n", command)
	return c.Run()
}
