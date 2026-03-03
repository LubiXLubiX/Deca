package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Alias for serve (unified dev server)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return serveCmd.RunE(cmd, args)
	},
}

var dbCreateCmd = &cobra.Command{
	Use:   "db:create",
	Short: "Create the configured database",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLubixCmd("db:create")
	},
}

var makeControllerCmd = &cobra.Command{
	Use:   "make:controller [name]",
	Short: "Generate a controller",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLubixCmdWithArgs("make:controller", args)
	},
}

var makeModelCmd = &cobra.Command{
	Use:   "make:model [name]",
	Short: "Generate a model",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLubixCmdWithArgs("make:model", args)
	},
}

var makeMigrationCmd = &cobra.Command{
	Use:   "make:migration [name]",
	Short: "Generate a migration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLubixCmdWithArgs("make:migration", args)
	},
}

func runLubixCmdWithArgs(command string, args []string) error {
	cwd, _ := os.Getwd()
	root := detectProjectRoot(cwd)
	if root == "" {
		return fmt.Errorf("not in a LubiX project")
	}

	cargs := append([]string{"lubix", command}, args...)
	c := exec.Command("php", cargs...)
	c.Dir = root
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	fmt.Printf("[+] [Deca] Running deca lubix %s %v...\n", command, args)
	return c.Run()
}
