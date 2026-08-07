package cmd

import (
	"fmt"
	"os"

	"github.com/nodecoda/auto-ingress/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  "Show, validate, and edit the auto-ingress configuration.",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	RunE:  runConfigShow,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	RunE:  runConfigValidate,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open configuration in editor",
	RunE:  runConfigEdit,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configEditCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	o := out()

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if jsonOutput {
		return o.JSON(cfg)
	}

	path := cfgFile
	if path == "" {
		path, _ = config.FindConfigFile()
	}

	o.Header("Configuration")
	o.KeyValue("File", path)
	o.KeyValue("Email", cfg.Gateway.Email)
	o.KeyValue("DNS", cfg.DNS.Provider)
	o.KeyValue("Sites", fmt.Sprintf("%d configured", len(cfg.Sites)))
	fmt.Println()

	for _, s := range cfg.Sites {
		fmt.Printf("  %s → %s\n", s.Domain, s.Upstream)
	}

	return nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	o := out()

	cfg, err := loadConfig()
	if err != nil {
		o.Error("Failed to load configuration: %v", err)
		return err
	}

	validator := config.NewValidator()
	errs := validator.Validate(cfg)

	if len(errs) == 0 {
		o.Success("Configuration is valid")
		return nil
	}

	o.Error("Configuration has %d error(s):", len(errs))
	for _, e := range errs {
		msg := fmt.Sprintf("  ✗ %s: %s", e.Path, e.Message)
		if e.Value != "" {
			msg += fmt.Sprintf(" (got: %q)", e.Value)
		}
		fmt.Fprintln(os.Stderr, msg)
	}

	return fmt.Errorf("configuration invalid")
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	path := cfgFile
	if path == "" {
		found, err := config.FindConfigFile()
		if err != nil {
			path = "auto-ingress.yaml"
		} else {
			path = found
		}
	}

	fmt.Printf("Run: %s %s\n", editor, path)
	return nil
}
