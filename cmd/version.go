package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("auto-ingress %s\n", version)
		fmt.Printf("  License:  MIT\n")
		fmt.Printf("  Homepage: https://github.com/nodecoda/auto-ingress\n")
	},
}
