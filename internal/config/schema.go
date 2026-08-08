package config

// Config is the top-level auto-ingress configuration
type Config struct {
    Version   string          `yaml:"version"`
    Gateway   GatewayConfig   `yaml:"gateway"`
    DNS       DNSConfig       `yaml:"dns"`
    Sites     []SiteConfig    `yaml:"sites"`
    Security  SecurityConfig  `yaml:"security"`
    Discovery DiscoveryConfig `yaml:"discovery"`
    Logging   LoggingConfig   `yaml:"logging"`
}

// GatewayConfig holds gateway server settings
type GatewayConfig struct {
    Email       string `yaml:"email"`
    HTTPBind    string `yaml:"http_bind"`
    HTTPSBind   string `yaml:"https_bind"`
    AdminBind   string `yaml:"admin_bind"`
    MemoryLimit string `yaml:"memory_limit"`
    PidsLimit   int    `yaml:"pids_limit"`
    DockerImage string `yaml:"docker_image"`
}

// DNSConfig holds DNS provider settings
type DNSConfig struct {
    Provider string `yaml:"provider"`
    EnvFile  string `yaml:"env_file"`
}

// SiteConfig defines a domain-to-upstream mapping
type SiteConfig struct {
    Domain   string            `yaml:"domain"`
    Upstream string            `yaml:"upstream"`
    Health   *HealthConfig     `yaml:"health,omitempty"`
    Headers  map[string]string `yaml:"headers,omitempty"`
}

// HealthConfig defines health check parameters
type HealthConfig struct {
    URI      string `yaml:"uri"`
    Interval string `yaml:"interval"`
    Timeout  string `yaml:"timeout"`
}

// SecurityConfig defines HTTP security headers
type SecurityConfig struct {
    HSTS               bool   `yaml:"hsts"`
    XFrameOptions      string `yaml:"x_frame_options"`
    ContentTypeNosniff bool   `yaml:"content_type_nosniff"`
    ReferrerPolicy     string `yaml:"referrer_policy"`
    HideServerHeader   bool   `yaml:"hide_server_header"`
}

// DiscoveryConfig defines Docker label-based service discovery
type DiscoveryConfig struct {
    Enabled      bool   `yaml:"enabled"`
    LabelPrefix  string `yaml:"label_prefix"`
    PollInterval string `yaml:"poll_interval"`
    Network      string `yaml:"network"`
}

// LoggingConfig defines logging settings
type LoggingConfig struct {
    AccessLog AccessLogConfig `yaml:"access"`
    Level     string          `yaml:"level"`
}

// AccessLogConfig defines access log output
type AccessLogConfig struct {
    Enabled bool   `yaml:"enabled"`
    Format  string `yaml:"format"`
    Output  string `yaml:"output"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
    return &Config{
        Version: "1",
        Gateway: GatewayConfig{
            HTTPBind:    "0.0.0.0:80",
            HTTPSBind:   "0.0.0.0:443",
            AdminBind:   "127.0.0.1:2019",
            MemoryLimit: "256m",
            PidsLimit:   256,
            DockerImage: "",
        },
        DNS: DNSConfig{
            Provider: "tencentcloud",
        },
        Security: SecurityConfig{
            HSTS:               true,
            XFrameOptions:      "SAMEORIGIN",
            ContentTypeNosniff: true,
            ReferrerPolicy:     "strict-origin-when-cross-origin",
            HideServerHeader:   true,
        },
        Discovery: DiscoveryConfig{
            LabelPrefix:  "auto-ingress",
            PollInterval: "30s",
        },
        Logging: LoggingConfig{
            AccessLog: AccessLogConfig{
                Enabled: true,
                Format:  "json",
                Output:  "stdout",
            },
            Level: "info",
        },
    }
}
