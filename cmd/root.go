package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "1.0.0"
)

var rootCmd = &cobra.Command{
	Use:   "deca",
	Short: "Deca CLI - The Professional Monorepo Toolchain",
	Long:  `Deca CLI is the ultimate tool for managing LubiX projects with a professional Flutter-like experience.`,
}

var lubixCmd = &cobra.Command{
	Use:   "lubix",
	Short: "LubiX framework commands",
}

func Execute() {
	rootCmd.AddCommand(createProjectCmd)
	rootCmd.AddCommand(lubixCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(versionCmd)

	// Add lubix subcommands
	lubixCmd.AddCommand(serveCmd)
	lubixCmd.AddCommand(devCmd)
	lubixCmd.AddCommand(migrateCmd)
	lubixCmd.AddCommand(rollbackCmd)
	lubixCmd.AddCommand(dbCreateCmd)
	lubixCmd.AddCommand(makeControllerCmd)
	lubixCmd.AddCommand(makeModelCmd)
	lubixCmd.AddCommand(makeMigrationCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print deca version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("deca version %s\n", version)
	},
}
