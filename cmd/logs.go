package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/nodecoda/auto-ingress/internal/docker"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View gateway logs",
	Long:  "Display gateway container logs.",
	RunE:  runLogs,
}

var (
	logsFollow bool
	logsSince  string
)

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "follow log output")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "show logs since (e.g. 1h, 30m)")
}

func runLogs(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	mgr, err := docker.NewManager()
	if err != nil {
		return fmt.Errorf("docker: %w", err)
	}
	defer mgr.Close()

	reader, err := mgr.Logs(ctx, logsFollow, logsSince)
	if err != nil {
		return fmt.Errorf("logs: %w", err)
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	if err != nil && err != io.EOF {
		return fmt.Errorf("read logs: %w", err)
	}

	return nil
}
