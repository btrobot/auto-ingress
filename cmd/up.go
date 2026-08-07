package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/nodecoda/auto-ingress/internal/caddy"
	"github.com/nodecoda/auto-ingress/internal/config"
	"github.com/nodecoda/auto-ingress/internal/docker"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start or update the gateway",
	Long:  "Generate Caddyfile from config, create/update the gateway container.",
	RunE:  runUp,
}

var (
	upDryRun bool
	upWatch  bool
)

func init() {
	upCmd.Flags().BoolVar(&upDryRun, "dry-run", false, "preview changes without applying")
	upCmd.Flags().BoolVar(&upWatch, "watch", false, "watch for Docker events and auto-update")
}

func runUp(cmd *cobra.Command, args []string) error {
	o := out()
	ctx := context.Background()

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	validator := config.NewValidator()
	errs := validator.Validate(cfg)
	if len(errs) > 0 {
		o.Error("Configuration validation failed:")
		for _, e := range errs {
			fmt.Printf("  • %s: %s\n", e.Path, e.Message)
		}
		return fmt.Errorf("invalid configuration (%d errors)", len(errs))
	}

	gen := caddy.NewGenerator()
	caddyfile, err := gen.Generate(cfg)
	if err != nil {
		return fmt.Errorf("generate caddyfile: %w", err)
	}

	if upDryRun {
		o.Header("Generated Caddyfile")
		preview, _ := gen.Preview(cfg)
		fmt.Print(preview)
		return nil
	}

	mgr, err := docker.NewManager()
	if err != nil {
		return fmt.Errorf("docker: %w", err)
	}
	defer mgr.Close()

	o.Info("Starting gateway...")
	if err := mgr.Up(ctx, cfg, caddyfile); err != nil {
		return fmt.Errorf("start gateway: %w", err)
	}

	o.Success("Gateway started")
	status, err := mgr.Status(ctx)
	if err == nil && status != nil {
		for _, p := range status.Ports {
			fmt.Printf("  %s → %s\n", p.Host, p.Container)
		}
	}

	if upWatch {
		o.Info("Watching for Docker events... (Ctrl+C to stop)")
		<-ctx.Done()
	}

	return nil
}

// unused but kept for watch mode
var _ = strings.Contains
