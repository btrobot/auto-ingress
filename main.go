package main

import (
	"fmt"
	"os"

	"github.com/nodecoda/auto-ingress/cmd"
)

// Version is set at build time via ldflags: -X github.com/nodecoda/auto-ingress/cmd.version=...
func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
