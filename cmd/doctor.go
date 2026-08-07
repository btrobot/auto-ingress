package cmd

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"time"

	"github.com/nodecoda/auto-ingress/internal/config"
	"github.com/nodecoda/auto-ingress/internal/docker"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose configuration and connectivity",
	Long:  "Run diagnostic checks on the auto-ingress configuration and gateway.",
	RunE:  runDoctor,
}

var doctorFix bool

func init() {
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "attempt automatic fixes")
}

type checkResult struct {
	Name    string
	Status  string // "pass", "warn", "fail"
	Message string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	o := out()
	ctx := context.Background()
	var results []checkResult

	// 1. Config checks
	cfg, cfgErr := loadConfig()
	if cfgErr != nil {
		results = append(results, checkResult{"Configuration", "fail", cfgErr.Error()})
	} else {
		results = append(results, checkResult{"Configuration", "pass", "auto-ingress.yaml loaded"})
		validator := config.NewValidator()
		errs := validator.Validate(cfg)
		if len(errs) == 0 {
			results = append(results, checkResult{"Config validation", "pass", "all fields valid"})
		} else {
			results = append(results, checkResult{"Config validation", "fail", fmt.Sprintf("%d error(s)", len(errs))})
		}
	}

	// 2. Docker checks
	if _, err := exec.LookPath("docker"); err != nil {
		results = append(results, checkResult{"Docker CLI", "fail", "docker not found in PATH"})
	} else {
		results = append(results, checkResult{"Docker CLI", "pass", "found"})
		if err := exec.Command("docker", "info").Run(); err != nil {
			results = append(results, checkResult{"Docker daemon", "fail", "cannot connect to Docker daemon"})
		} else {
			results = append(results, checkResult{"Docker daemon", "pass", "running"})
		}
	}

	// 3. Network checks
	if cfg != nil && cfg.Discovery.Network != "" {
		if err := exec.Command("docker", "network", "inspect", cfg.Discovery.Network).Run(); err != nil {
			results = append(results, checkResult{"Network " + cfg.Discovery.Network, "warn", "not found (will be created)"})
		} else {
			results = append(results, checkResult{"Network " + cfg.Discovery.Network, "pass", "exists"})
		}
	}

	// 4. Gateway container check
	mgr, err := docker.NewManager()
	if err == nil {
		defer mgr.Close()
		status, err := mgr.Status(ctx)
		if err != nil || status == nil || !status.Running {
			results = append(results, checkResult{"Gateway container", "fail", "not running"})
		} else {
			results = append(results, checkResult{"Gateway container", "pass", fmt.Sprintf("running (up %s)", status.Uptime)})
		}
	}

	// 5. Port availability checks
	if cfg != nil {
		for _, bind := range []struct{ name, addr string }{
			{"HTTP", cfg.Gateway.HTTPBind},
			{"HTTPS", cfg.Gateway.HTTPSBind},
		} {
			if bind.addr == "" {
				continue
			}
			_, port, err := net.SplitHostPort(bind.addr)
			if err != nil {
				results = append(results, checkResult{bind.name + " bind", "warn", "invalid address format"})
				continue
			}
			ln, err := net.Listen("tcp", ":"+port)
			if err != nil {
				results = append(results, checkResult{bind.name + " port " + port, "warn", "port in use"})
			} else {
				ln.Close()
				results = append(results, checkResult{bind.name + " port " + port, "pass", "available"})
			}
		}
	}

	// 6. Upstream connectivity checks
	if cfg != nil {
		for _, site := range cfg.Sites {
			conn, err := net.DialTimeout("tcp", site.Upstream, 3*time.Second)
			if err != nil {
				results = append(results, checkResult{"Upstream " + site.Upstream, "fail", "unreachable"})
			} else {
				conn.Close()
				results = append(results, checkResult{"Upstream " + site.Upstream, "pass", "reachable"})
			}
		}
	}

	// Print results
	o.Header("Diagnostic Results")
	fmt.Println()

	passed, warned, failed := 0, 0, 0
	for _, r := range results {
		icon := "  "
		switch r.Status {
		case "pass":
			icon = "✓"
			passed++
		case "warn":
			icon = "⚠"
			warned++
		case "fail":
			icon = "✗"
			failed++
		}
		fmt.Printf("  %s %s: %s\n", icon, r.Name, r.Message)
	}

	fmt.Println()
	summary := fmt.Sprintf("%d passed", passed)
	if warned > 0 {
		summary += fmt.Sprintf(", %d warnings", warned)
	}
	if failed > 0 {
		summary += fmt.Sprintf(", %d errors", failed)
	}
	fmt.Printf("Summary: %s\n", summary)

	if failed > 0 {
		return fmt.Errorf("doctor found %d errors", failed)
	}
	return nil
}
