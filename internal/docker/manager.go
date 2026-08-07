package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nodecoda/auto-ingress/internal/config"
)

const (
	managedLabel = "auto-ingress.managed"
	containerName = "auto-ingress-gateway"
)

// GatewayStatus represents the current gateway state
type GatewayStatus struct {
	Running     bool          `json:"running"`
	ContainerID string        `json:"container_id"`
	Image       string        `json:"image"`
	Uptime      string        `json:"uptime"`
	Ports       []PortMapping `json:"ports"`
}

// PortMapping represents a port binding
type PortMapping struct {
	Host      string `json:"host"`
	Container string `json:"container"`
	Protocol  string `json:"protocol"`
}

// Manager manages the auto-ingress Docker container
type Manager struct{}

// NewManager creates a new Docker manager
func NewManager() (*Manager, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker not found in PATH")
	}
	// Check Docker daemon is running
	if err := exec.Command("docker", "info").Run(); err != nil {
		return nil, fmt.Errorf("docker daemon is not running")
	}
	return &Manager{}, nil
}

// Close is a no-op for the CLI-based manager
func (m *Manager) Close() error { return nil }

// Up creates or recreates the gateway container
func (m *Manager) Up(ctx context.Context, cfg *config.Config, caddyfile []byte) error {
	// Stop existing
	m.Down(ctx)

	// Ensure network
	if cfg.Discovery.Network != "" {
		m.EnsureNetwork(ctx, cfg.Discovery.Network)
	}

	// Write Caddyfile
	caddyDir := "/tmp/auto-ingress"
	os.MkdirAll(caddyDir, 0755)
	caddyfilePath := caddyDir + "/Caddyfile"
	if err := os.WriteFile(caddyfilePath, caddyfile, 0644); err != nil {
		return fmt.Errorf("write caddyfile: %w", err)
	}

	// Build docker run args
	image := cfg.Gateway.DockerImage
	if image == "" {
		image = "caddy:2.10.2-alpine"
	}

	httpBind := cfg.Gateway.HTTPBind
	if httpBind == "" {
		httpBind = "0.0.0.0:80"
	}
	httpsBind := cfg.Gateway.HTTPSBind
	if httpsBind == "" {
		httpsBind = "0.0.0.0:443"
	}
	adminBind := cfg.Gateway.AdminBind
	if adminBind == "" {
		adminBind = "127.0.0.1:2019"
	}

	args := []string{
		"run", "-d",
		"--name", containerName,
		"--label", managedLabel + "=true",
		"--restart", "unless-stopped",
		"--cap-add", "NET_BIND_SERVICE",
		"--security-opt", "no-new-privileges:true",
		"--read-only",
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=16m",
		"-p", httpBind + ":80",
		"-p", httpsBind + ":443",
		"-p", httpsBind + ":443/udp",
		"-p", adminBind + ":2019",
		"-v", caddyfilePath + ":/etc/caddy/Caddyfile:ro",
		"-v", "auto-ingress-data:/data",
		"-v", "auto-ingress-config:/config",
	}

	// Memory limit
	memLimit := cfg.Gateway.MemoryLimit
	if memLimit == "" {
		memLimit = "256m"
	}
	args = append(args, "--memory", memLimit)

	// PID limit
	if cfg.Gateway.PidsLimit > 0 {
		args = append(args, "--pids-limit", fmt.Sprintf("%d", cfg.Gateway.PidsLimit))
	}

	// Network
	if cfg.Discovery.Network != "" {
		args = append(args, "--network", cfg.Discovery.Network)
	}

	// DNS env vars
	args = append(args, "-e", "ACME_EMAIL="+cfg.Gateway.Email)
	if cfg.DNS.EnvFile != "" {
		data, err := os.ReadFile(cfg.DNS.EnvFile)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					args = append(args, "-e", line)
				}
			}
		}
	}

	args = append(args, image)
	args = append(args, "caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile")

	// Pull image first
	pull := exec.CommandContext(ctx, "docker", "pull", image)
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	pull.Run() // best effort

	// Run container
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker run: %w\n%s", err, string(output))
	}

	return nil
}

// Down stops and removes the gateway container
func (m *Manager) Down(ctx context.Context) error {
	exec.CommandContext(ctx, "docker", "stop", containerName).Run()
	exec.CommandContext(ctx, "docker", "rm", containerName).Run()
	return nil
}

// Status returns current gateway status
func (m *Manager) Status(ctx context.Context) (*GatewayStatus, error) {
	out, err := exec.CommandContext(ctx, "docker", "inspect",
		"--format", `{"id":"{{.Id}}","image":"{{.Config.Image}}","running":{{.State.Running}},"started":"{{.State.StartedAt}}"}`,
		containerName).Output()
	if err != nil {
		return nil, nil // container doesn't exist
	}

	var raw struct {
		ID      string `json:"id"`
		Image   string `json:"image"`
		Running bool   `json:"running"`
		Started string `json:"started"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse inspect: %w", err)
	}

	status := &GatewayStatus{
		Running:     raw.Running,
		ContainerID: raw.ID,
		Image:       raw.Image,
	}

	if raw.Started != "" && raw.Running {
		started, err := time.Parse(time.RFC3339Nano, raw.Started)
		if err == nil {
			status.Uptime = time.Since(started).Truncate(time.Second).String()
		}
	}

	// Get port mappings
	portOut, _ := exec.CommandContext(ctx, "docker", "port", containerName).Output()
	for _, line := range strings.Split(strings.TrimSpace(string(portOut)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " -> ", 2)
		if len(parts) == 2 {
			status.Ports = append(status.Ports, PortMapping{
				Host:      strings.TrimSpace(parts[1]),
				Container: strings.TrimSpace(parts[0]),
			})
		}
	}

	return status, nil
}

// Logs returns a reader for container logs
func (m *Manager) Logs(ctx context.Context, follow bool, since string) (io.ReadCloser, error) {
	args := []string{"logs", "--timestamps"}
	if follow {
		args = append(args, "-f")
	}
	if since != "" {
		args = append(args, "--since", since)
	}
	args = append(args, containerName)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stderr = cmd.Stdout // merge stderr into stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return stdout, nil
}

// Restart restarts the gateway container
func (m *Manager) Restart(ctx context.Context) error {
	out, err := exec.CommandContext(ctx, "docker", "restart", containerName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("restart: %w\n%s", err, string(out))
	}
	return nil
}

// EnsureNetwork creates the Docker network if it doesn't exist
func (m *Manager) EnsureNetwork(ctx context.Context, name string) error {
	// Check if exists
	check := exec.CommandContext(ctx, "docker", "network", "inspect", name)
	if check.Run() == nil {
		return nil
	}
	// Create
	out, err := exec.CommandContext(ctx, "docker", "network", "create", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("create network %s: %w\n%s", name, err, string(out))
	}
	return nil
}
