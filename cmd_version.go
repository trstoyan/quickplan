package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of QuickPlan",
	Long:  `All software has versions. This is QuickPlan's`,
	Run: func(cmd *cobra.Command, args []string) {
		if globalJSON {
			fmt.Printf("{\"version\": \"%s\", \"build_date\": \"%s\"}\n", version, "2026-02-24")
			os.Exit(0)
		}
		fmt.Printf("QuickPlan version %s\n", version)
	},
}
