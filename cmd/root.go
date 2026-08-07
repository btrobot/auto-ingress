package cmd

import (
	"fmt"
	"os"

	"github.com/nodecoda/auto-ingress/internal/config"
	"github.com/nodecoda/auto-ingress/pkg/output"
	"github.com/spf13/cobra"
)

var (
	version    = "dev"
	cfgFile    string
	jsonOutput bool
	noColor    bool
)

func SetVersion(v string) { version = v }

var rootCmd = &cobra.Command{
	Use:   "auto-ingress",
	Short: "Automatic HTTPS reverse proxy gateway",
	Long: `Auto Ingress — Domain + Service + Auto HTTPS Gateway

A CLI tool that manages a Caddy-based reverse proxy with automatic
Let's Encrypt certificate provisioning via DNS-01 challenges.

Quick start:
  auto-ingress init                    Interactive configuration wizard
  auto-ingress site add example.com web:3000
  auto-ingress up                      Start the gateway
  auto-ingress status                  Check gateway health`,
	SilenceUsage: true,
}

func Execute() error { return rootCmd.Execute() }

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./auto-ingress.yaml)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(siteCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(certCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(versionCmd)
}

// loadConfig loads configuration from the configured path
func loadConfig() (*config.Config, error) {
	path := cfgFile
	if path == "" {
		found, err := config.FindConfigFile()
		if err != nil {
			return nil, fmt.Errorf("no config file found (run 'auto-ingress init' first): %w", err)
		}
		path = found
	}
	loader := config.NewLoader(path)
	return loader.Load()
}

// out returns a configured output helper
func out() *output.Output {
	o := output.Stdout()
	o.SetJSONMode(jsonOutput)
	o.SetNoColor(noColor)
	return o
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
