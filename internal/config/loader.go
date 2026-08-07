package config

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "gopkg.in/yaml.v3"
)

// Loader loads configuration from files and environment
type Loader struct {
    configPath string
}

// NewLoader creates a loader for the given config file path
func NewLoader(configPath string) *Loader {
    return &Loader{configPath: configPath}
}

// Load reads and merges configuration from file + env overrides
func (l *Loader) Load() (*Config, error) {
    data, err := os.ReadFile(l.configPath)
    if err != nil {
        return nil, fmt.Errorf("read config %s: %w", l.configPath, err)
    }
    cfg, err := LoadFromBytes(data)
    if err != nil {
        return nil, err
    }
    applyDefaults(cfg)
    applyEnvOverrides(cfg)
    return cfg, nil
}

// LoadFromBytes parses config from YAML bytes
func LoadFromBytes(data []byte) (*Config, error) {
    cfg := &Config{}
    if err := yaml.Unmarshal(data, cfg); err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }
    return cfg, nil
}

// FindConfigFile searches for config file in standard locations
func FindConfigFile() (string, error) {
    candidates := []string{
        "auto-ingress.yaml",
        "auto-ingress.yml",
    }

    // Also check user config dir
    if home, err := os.UserHomeDir(); err == nil {
        candidates = append(candidates,
            filepath.Join(home, ".config", "auto-ingress", "config.yaml"),
            filepath.Join(home, ".config", "auto-ingress", "config.yml"),
        )
    }

    // System config
    candidates = append(candidates,
        "/etc/auto-ingress/config.yaml",
        "/etc/auto-ingress/config.yml",
    )

    for _, c := range candidates {
        if _, err := os.Stat(c); err == nil {
            return c, nil
        }
    }
    return "", fmt.Errorf("no config file found (searched: %s)", strings.Join(candidates, ", "))
}

// applyDefaults fills in missing fields with defaults
func applyDefaults(cfg *Config) {
    def := DefaultConfig()

    if cfg.Version == "" {
        cfg.Version = def.Version
    }
    if cfg.Gateway.HTTPBind == "" {
        cfg.Gateway.HTTPBind = def.Gateway.HTTPBind
    }
    if cfg.Gateway.HTTPSBind == "" {
        cfg.Gateway.HTTPSBind = def.Gateway.HTTPSBind
    }
    if cfg.Gateway.AdminBind == "" {
        cfg.Gateway.AdminBind = def.Gateway.AdminBind
    }
    if cfg.Gateway.MemoryLimit == "" {
        cfg.Gateway.MemoryLimit = def.Gateway.MemoryLimit
    }
    if cfg.Gateway.PidsLimit == 0 {
        cfg.Gateway.PidsLimit = def.Gateway.PidsLimit
    }
    if cfg.Gateway.DockerImage == "" {
        cfg.Gateway.DockerImage = def.Gateway.DockerImage
    }
    if cfg.DNS.Provider == "" {
        cfg.DNS.Provider = def.DNS.Provider
    }
    if cfg.Discovery.LabelPrefix == "" {
        cfg.Discovery.LabelPrefix = def.Discovery.LabelPrefix
    }
    if cfg.Discovery.PollInterval == "" {
        cfg.Discovery.PollInterval = def.Discovery.PollInterval
    }
    if cfg.Logging.Level == "" {
        cfg.Logging.Level = def.Logging.Level
    }
    if cfg.Logging.AccessLog.Format == "" {
        cfg.Logging.AccessLog.Format = def.Logging.AccessLog.Format
    }
    if cfg.Logging.AccessLog.Output == "" {
        cfg.Logging.AccessLog.Output = def.Logging.AccessLog.Output
    }
    if !cfg.Security.HSTS && !cfg.Security.ContentTypeNosniff {
        // Only apply security defaults if security block is empty
        cfg.Security = def.Security
    }
    if cfg.Security.ReferrerPolicy == "" {
        cfg.Security.ReferrerPolicy = def.Security.ReferrerPolicy
    }
    if cfg.Security.XFrameOptions == "" {
        cfg.Security.XFrameOptions = def.Security.XFrameOptions
    }

    // Apply health defaults to sites
    for i := range cfg.Sites {
        if cfg.Sites[i].Health == nil {
            cfg.Sites[i].Health = &HealthConfig{
                URI:      "/health",
                Interval: "30s",
                Timeout:  "5s",
            }
        }
    }
}

// applyEnvOverrides overrides config values from environment variables
func applyEnvOverrides(cfg *Config) {
    if v := os.Getenv("AUTO_INGRESS_EMAIL"); v != "" {
        cfg.Gateway.Email = v
    }
    if v := os.Getenv("AUTO_INGRESS_DNS_PROVIDER"); v != "" {
        cfg.DNS.Provider = v
    }
    if v := os.Getenv("AUTO_INGRESS_HTTP_BIND"); v != "" {
        cfg.Gateway.HTTPBind = v
    }
    if v := os.Getenv("AUTO_INGRESS_HTTPS_BIND"); v != "" {
        cfg.Gateway.HTTPSBind = v
    }
    if v := os.Getenv("AUTO_INGRESS_ADMIN_BIND"); v != "" {
        cfg.Gateway.AdminBind = v
    }
    if v := os.Getenv("AUTO_INGRESS_MEMORY_LIMIT"); v != "" {
        cfg.Gateway.MemoryLimit = v
    }
    if v := os.Getenv("AUTO_INGRESS_DOCKER_IMAGE"); v != "" {
        cfg.Gateway.DockerImage = v
    }
    if v := os.Getenv("AUTO_INGRESS_DNS_ENV_FILE"); v != "" {
        cfg.DNS.EnvFile = v
    }
}
