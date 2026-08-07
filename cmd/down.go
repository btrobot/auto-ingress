package cmd

import (
	"context"
	"fmt"

	"github.com/nodecoda/auto-ingress/internal/docker"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop the gateway",
	Long:  "Stop and remove the gateway container.",
	RunE:  runDown,
}

func runDown(cmd *cobra.Command, args []string) error {
	o := out()
	ctx := context.Background()

	mgr, err := docker.NewManager()
	if err != nil {
		return fmt.Errorf("docker: %w", err)
	}
	defer mgr.Close()

	o.Info("Stopping gateway...")
	if err := mgr.Down(ctx); err != nil {
		return fmt.Errorf("stop gateway: %w", err)
	}

	o.Success("Gateway stopped")
	return nil
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the gateway",
	RunE: func(cmd *cobra.Command, args []string) error {
		o := out()
		ctx := context.Background()

		mgr, err := docker.NewManager()
		if err != nil {
			return fmt.Errorf("docker: %w", err)
		}
		defer mgr.Close()

		o.Info("Restarting gateway...")
		if err := mgr.Restart(ctx); err != nil {
			return fmt.Errorf("restart gateway: %w", err)
		}

		o.Success("Gateway restarted")
		return nil
	},
}
