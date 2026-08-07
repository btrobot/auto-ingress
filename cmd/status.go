package cmd

import (
	"context"
	"fmt"

	"github.com/nodecoda/auto-ingress/internal/docker"
	"github.com/nodecoda/auto-ingress/pkg/output"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show gateway status",
	Long:  "Display current gateway status including sites and resources.",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	o := out()
	ctx := context.Background()

	mgr, err := docker.NewManager()
	if err != nil {
		return fmt.Errorf("docker: %w", err)
	}
	defer mgr.Close()

	status, err := mgr.Status(ctx)
	if err != nil {
		return fmt.Errorf("get status: %w", err)
	}

	if jsonOutput {
		return o.JSON(status)
	}

	if status == nil || !status.Running {
		o.Error("Gateway is not running")
		fmt.Println("  Run 'auto-ingress up' to start it.")
		return nil
	}

	cfg, _ := loadConfig()

	o.Header("Auto Ingress Gateway")
	fmt.Println()

	state := "running"
	fmt.Printf("  Status:    %s %s\n", output.StatusIndicator(state), state)
	if status.Uptime != "" {
		fmt.Printf("  Uptime:    %s\n", status.Uptime)
	}
	if status.Image != "" {
		fmt.Printf("  Image:     %s\n", status.Image)
	}
	if len(status.ContainerID) >= 12 {
		fmt.Printf("  Container: %s\n", status.ContainerID[:12])
	}
	fmt.Println()

	if cfg != nil && len(cfg.Sites) > 0 {
		o.Header(fmt.Sprintf("Sites (%d)", len(cfg.Sites)))
		headers := []string{"DOMAIN", "UPSTREAM", "STATUS"}
		var rows [][]string
		for _, s := range cfg.Sites {
			rows = append(rows, []string{
				s.Domain,
				s.Upstream,
				output.StatusIndicator("running"),
			})
		}
		o.Table(headers, rows)
		fmt.Println()
	}

	if len(status.Ports) > 0 {
		o.Header("Ports")
		for _, p := range status.Ports {
			fmt.Printf("  %s → %s\n", p.Host, p.Container)
		}
	}

	return nil
}
