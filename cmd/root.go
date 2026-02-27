package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version   = "0.1.0"
	commit    = "dev"
	buildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "lubix",
	Short: "LubiX Software CLI",
	Long:  "LubiX Software CLI - create projects and run the LubiX full-stack dev environment.",
}

func Execute() {
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, buildDate)
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.VersionTemplate()
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(createProjectCmd)
	rootCmd.AddCommand(runCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
