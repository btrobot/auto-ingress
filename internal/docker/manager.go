package docker

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	units "github.com/docker/go-units"
	"github.com/nodecoda/auto-ingress/internal/config"
)

const (
	managedLabel  = "auto-ingress.managed"
	containerName = "auto-ingress-caddy"
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

// Manager manages the auto-ingress Docker container via the Docker SDK
type Manager struct {
	cli *client.Client
}

// NewManager creates a new Docker manager using the Docker Go SDK.
// It pings the daemon to verify connectivity.
func NewManager() (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("docker daemon is not running: %w", err)
	}
	return &Manager{cli: cli}, nil
}

// Close releases the Docker client resources
func (m *Manager) Close() error {
	return m.cli.Close()
}

// Up creates or recreates the gateway container
func (m *Manager) Up(ctx context.Context, cfg *config.Config, caddyfile []byte) error {
	// Stop existing
	m.Down(ctx)

	// Ensure network
	if cfg.Discovery.Network != "" {
		if err := m.EnsureNetwork(ctx, cfg.Discovery.Network); err != nil {
			return err
		}
	}

	// Write Caddyfile to host
	caddyDir := "/srv/auto-ingress"
	if err := os.MkdirAll(caddyDir, 0755); err != nil {
		return fmt.Errorf("create caddy dir: %w", err)
	}
	caddyfilePath := caddyDir + "/Caddyfile"
	if err := os.WriteFile(caddyfilePath, caddyfile, 0644); err != nil {
		return fmt.Errorf("write caddyfile: %w", err)
	}

	imageName := cfg.Gateway.DockerImage
	if imageName == "" {
		imageName = os.Getenv("SELF_IMAGE")
	}
	if imageName == "" {
		imageName = "caddy:2.10.2-alpine"
	}

	// ── Build port bindings ──────────────────────────────────────
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

	httpHost, httpPort := splitBind(httpBind)
	httpsHost, httpsPort := splitBind(httpsBind)
	adminHost, adminPort := splitBind(adminBind)

	portSet := nat.PortSet{
		"80/tcp":   {},
		"443/tcp":  {},
		"443/udp":  {},
		"2019/tcp": {},
	}
	portMap := nat.PortMap{
		"80/tcp":   {{HostIP: httpHost, HostPort: httpPort}},
		"443/tcp":  {{HostIP: httpsHost, HostPort: httpsPort}},
		"443/udp":  {{HostIP: httpsHost, HostPort: httpsPort}},
		"2019/tcp": {{HostIP: adminHost, HostPort: adminPort}},
	}

	// ── Build environment variables ──────────────────────────────
	env := []string{"ACME_EMAIL=" + cfg.Gateway.Email}
	if cfg.DNS.EnvFile != "" {
		data, err := os.ReadFile(cfg.DNS.EnvFile)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					env = append(env, line)
				}
			}
		}
	}

	// ── Container config ─────────────────────────────────────────
	containerCfg := &container.Config{
		Image:        imageName,
		Labels:       map[string]string{managedLabel: "true"},
		ExposedPorts: portSet,
		Env:          env,
		Entrypoint:   []string{"caddy"},
		Cmd:          []string{"run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"},
	}

	// ── Host config ──────────────────────────────────────────────
	memLimit := cfg.Gateway.MemoryLimit
	if memLimit == "" {
		memLimit = "256m"
	}
	memBytes, _ := units.FromHumanSize(memLimit)

	var pidsLimit *int64
	if cfg.Gateway.PidsLimit > 0 {
		v := int64(cfg.Gateway.PidsLimit)
		pidsLimit = &v
	}

	hostCfg := &container.HostConfig{
		PortBindings: portMap,
		Binds: []string{
			caddyfilePath + ":/etc/caddy/Caddyfile:ro",
			"auto-ingress-data:/data",
			"auto-ingress-config:/config",
		},
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		CapAdd:         []string{"NET_BIND_SERVICE"},
		SecurityOpt:    []string{"no-new-privileges:true"},
		ReadonlyRootfs: true,
		Tmpfs: map[string]string{
			"/tmp": "rw,noexec,nosuid,nodev,size=16m",
		},
		Resources: container.Resources{
			Memory:    memBytes,
			PidsLimit: pidsLimit,
		},
	}

	// ── Network config ───────────────────────────────────────────
	networkCfg := &network.NetworkingConfig{}
	if cfg.Discovery.Network != "" {
		networkCfg.EndpointsConfig = map[string]*network.EndpointSettings{
			cfg.Discovery.Network: {},
		}
	}

	// ── Create and start container ───────────────────────────────
	createResp, err := m.cli.ContainerCreate(ctx, containerCfg, hostCfg, networkCfg, nil, containerName)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	if err := m.cli.ContainerStart(ctx, createResp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	return nil
}

// Down stops and removes the gateway container
func (m *Manager) Down(ctx context.Context) error {
	timeout := 10
	_ = m.cli.ContainerStop(ctx, containerName, container.StopOptions{Timeout: &timeout})
	_ = m.cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
	return nil
}

// Status returns current gateway status
func (m *Manager) Status(ctx context.Context) (*GatewayStatus, error) {
	inspect, err := m.cli.ContainerInspect(ctx, containerName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	status := &GatewayStatus{
		Running:     inspect.State != nil && inspect.State.Running,
		ContainerID: inspect.ID,
		Image:       inspect.Config.Image,
	}

	if status.Running && inspect.State.StartedAt != "" {
		started, err := time.Parse(time.RFC3339Nano, inspect.State.StartedAt)
		if err == nil {
			status.Uptime = time.Since(started).Truncate(time.Second).String()
		}
	}

	if inspect.NetworkSettings != nil {
		for containerPort, bindings := range inspect.NetworkSettings.Ports {
			for _, b := range bindings {
				status.Ports = append(status.Ports, PortMapping{
					Host:      b.HostIP + ":" + b.HostPort,
					Container: string(containerPort),
					Protocol:  containerPort.Proto(),
				})
			}
		}
	}

	return status, nil
}

// Logs returns a reader for container logs
func (m *Manager) Logs(ctx context.Context, follow bool, since string) (io.ReadCloser, error) {
	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     follow,
	}
	if since != "" {
		opts.Since = since
	}
	reader, err := m.cli.ContainerLogs(ctx, containerName, opts)
	if err != nil {
		return nil, fmt.Errorf("container logs: %w", err)
	}
	return reader, nil
}

// Restart restarts the gateway container
func (m *Manager) Restart(ctx context.Context) error {
	timeout := 10
	if err := m.cli.ContainerRestart(ctx, containerName, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	return nil
}

// EnsureNetwork creates the Docker network if it doesn't exist
func (m *Manager) EnsureNetwork(ctx context.Context, name string) error {
	if m.HasNetwork(ctx, name) {
		return nil
	}
	_, err := m.cli.NetworkCreate(ctx, name, network.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create network %s: %w", name, err)
	}
	return nil
}

// HasNetwork checks whether a Docker network with the given name exists
func (m *Manager) HasNetwork(ctx context.Context, name string) bool {
	_, err := m.cli.NetworkInspect(ctx, name, network.InspectOptions{})
	return err == nil
}

// ── helpers ──────────────────────────────────────────────────────

// splitBind splits a "host:port" address into host and port.
// Uses net.SplitHostPort for correct IPv6 handling.
func splitBind(addr string) (string, string) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "0.0.0.0", addr
	}
	if host == "" {
		host = "0.0.0.0"
	}
	return host, port
}
