package config

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

// ValidationError represents a single validation error
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// Error implements the error interface
func (e ValidationError) Error() string {
	if e.Value != "" {
		return e.Path + ": " + e.Message + " (got: " + e.Value + ")"
	}
	return e.Path + ": " + e.Message
}

// Validator validates configuration
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// Validate checks the config and returns all validation errors
func (v *Validator) Validate(cfg *Config) []ValidationError {
	var errs []ValidationError
	errs = append(errs, v.validateGateway(cfg)...)
	errs = append(errs, v.validateDNS(cfg)...)
	errs = append(errs, v.validateSites(cfg)...)
	errs = append(errs, v.validateSecurity(cfg)...)
	errs = append(errs, v.validateDiscovery(cfg)...)
	return errs
}

func (v *Validator) validateGateway(cfg *Config) []ValidationError {
	var errs []ValidationError

	if cfg.Gateway.Email == "" {
		errs = append(errs, ValidationError{Path: "gateway.email", Message: "ACME email is required"})
	} else if !strings.Contains(cfg.Gateway.Email, "@") {
		errs = append(errs, ValidationError{Path: "gateway.email", Message: "invalid email format", Value: cfg.Gateway.Email})
	}

	for _, pair := range []struct {
		path, val string
	}{
		{"gateway.http_bind", cfg.Gateway.HTTPBind},
		{"gateway.https_bind", cfg.Gateway.HTTPSBind},
		{"gateway.admin_bind", cfg.Gateway.AdminBind},
	} {
		if pair.val != "" && !isValidBind(pair.val) {
			errs = append(errs, ValidationError{Path: pair.path, Message: "must be host:port format", Value: pair.val})
		}
	}

	if cfg.Gateway.PidsLimit < 0 {
		errs = append(errs, ValidationError{Path: "gateway.pids_limit", Message: "must be non-negative"})
	}

	return errs
}

func (v *Validator) validateDNS(cfg *Config) []ValidationError {
	var errs []ValidationError
	validProviders := map[string]bool{
		"tencentcloud": true, "aliyun": true, "cloudflare": true,
		"route53": true, "digitalocean": true, "googlecloud": true,
	}
	if cfg.DNS.Provider != "" && !validProviders[cfg.DNS.Provider] {
		errs = append(errs, ValidationError{
			Path:    "dns.provider",
			Message: "unsupported provider (valid: tencentcloud, aliyun, cloudflare, route53, digitalocean, googlecloud)",
			Value:   cfg.DNS.Provider,
		})
	}
	return errs
}

func (v *Validator) validateSites(cfg *Config) []ValidationError {
	var errs []ValidationError

	if len(cfg.Sites) == 0 && !cfg.Discovery.Enabled {
		errs = append(errs, ValidationError{Path: "sites", Message: "at least one site required when discovery is disabled"})
	}

	domains := make(map[string]bool)
	for i, site := range cfg.Sites {
		p := func(field string) string { return fmt.Sprintf("sites[%d].%s", i, field) }

		if site.Domain == "" {
			errs = append(errs, ValidationError{Path: p("domain"), Message: "domain is required"})
		} else if !isValidDomain(site.Domain) {
			errs = append(errs, ValidationError{Path: p("domain"), Message: "invalid domain format", Value: site.Domain})
		} else if domains[site.Domain] {
			errs = append(errs, ValidationError{Path: p("domain"), Message: "duplicate domain", Value: site.Domain})
		}
		domains[site.Domain] = true

		if site.Upstream == "" {
			errs = append(errs, ValidationError{Path: p("upstream"), Message: "upstream is required"})
		} else if !isValidUpstream(site.Upstream) {
			errs = append(errs, ValidationError{Path: p("upstream"), Message: "upstream must be host:port format", Value: site.Upstream})
		}

		if site.Health != nil {
			if site.Health.Interval != "" && !isValidDuration(site.Health.Interval) {
				errs = append(errs, ValidationError{Path: p("health.interval"), Message: "invalid duration", Value: site.Health.Interval})
			}
			if site.Health.Timeout != "" && !isValidDuration(site.Health.Timeout) {
				errs = append(errs, ValidationError{Path: p("health.timeout"), Message: "invalid duration", Value: site.Health.Timeout})
			}
		}
	}
	return errs
}

func (v *Validator) validateSecurity(cfg *Config) []ValidationError {
	var errs []ValidationError
	if cfg.Security.XFrameOptions != "" && cfg.Security.XFrameOptions != "DENY" && cfg.Security.XFrameOptions != "SAMEORIGIN" {
		errs = append(errs, ValidationError{Path: "security.x_frame_options", Message: "must be DENY or SAMEORIGIN", Value: cfg.Security.XFrameOptions})
	}
	return errs
}

func (v *Validator) validateDiscovery(cfg *Config) []ValidationError {
	var errs []ValidationError
	if cfg.Discovery.Enabled && cfg.Discovery.Network == "" {
		errs = append(errs, ValidationError{Path: "discovery.network", Message: "network is required when discovery is enabled"})
	}
	if cfg.Discovery.PollInterval != "" && !isValidDuration(cfg.Discovery.PollInterval) {
		errs = append(errs, ValidationError{Path: "discovery.poll_interval", Message: "invalid duration format", Value: cfg.Discovery.PollInterval})
	}
	return errs
}

var domainRegex = regexp.MustCompile(`^(\*\.)?[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$`)

func isValidDomain(domain string) bool {
	return domainRegex.MatchString(domain)
}

func isValidUpstream(upstream string) bool {
	host, port, err := net.SplitHostPort(upstream)
	return err == nil && host != "" && port != ""
}

func isValidBind(addr string) bool {
	_, _, err := net.SplitHostPort(addr)
	return err == nil
}

func isValidDuration(s string) bool {
	_, err := time.ParseDuration(s)
	return err == nil
}
