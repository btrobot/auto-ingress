package cmd

import (
	"fmt"
	"os"

	"github.com/nodecoda/auto-ingress/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var siteCmd = &cobra.Command{
	Use:   "site",
	Short: "Manage sites",
	Long:  "Add, remove, and list configured sites.",
}

var siteAddCmd = &cobra.Command{
	Use:   "add <domain> <upstream>",
	Short: "Add a site",
	Long:  "Add a new site mapping. Upstream format: host:port",
	Args:  cobra.ExactArgs(2),
	RunE:  runSiteAdd,
}

var siteRemoveCmd = &cobra.Command{
	Use:   "remove <domain>",
	Short: "Remove a site",
	Args:  cobra.ExactArgs(1),
	RunE:  runSiteRemove,
}

var siteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sites",
	RunE:  runSiteList,
}

func init() {
	siteCmd.AddCommand(siteAddCmd)
	siteCmd.AddCommand(siteRemoveCmd)
	siteCmd.AddCommand(siteListCmd)
}

func runSiteAdd(cmd *cobra.Command, args []string) error {
	o := out()
	domain := args[0]
	upstream := args[1]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	for _, s := range cfg.Sites {
		if s.Domain == domain {
			return fmt.Errorf("site %s already exists (upstream: %s)", domain, s.Upstream)
		}
	}

	cfg.Sites = append(cfg.Sites, config.SiteConfig{
		Domain:   domain,
		Upstream: upstream,
	})

	if err := saveConfig(cfg); err != nil {
		return err
	}

	o.Success("Site added: %s → %s", domain, upstream)
	return nil
}

func runSiteRemove(cmd *cobra.Command, args []string) error {
	o := out()
	domain := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	found := false
	var sites []config.SiteConfig
	for _, s := range cfg.Sites {
		if s.Domain == domain {
			found = true
			continue
		}
		sites = append(sites, s)
	}

	if !found {
		return fmt.Errorf("site %s not found", domain)
	}

	cfg.Sites = sites
	if err := saveConfig(cfg); err != nil {
		return err
	}

	o.Success("Site removed: %s", domain)
	return nil
}

func runSiteList(cmd *cobra.Command, args []string) error {
	o := out()

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if len(cfg.Sites) == 0 {
		o.Info("No sites configured. Use 'auto-ingress site add' to add one.")
		return nil
	}

	if jsonOutput {
		return o.JSON(cfg.Sites)
	}

	headers := []string{"DOMAIN", "UPSTREAM", "HEALTH"}
	var rows [][]string
	for _, s := range cfg.Sites {
		health := "/health (30s)"
		if s.Health != nil {
			health = fmt.Sprintf("%s (%s)", s.Health.URI, s.Health.Interval)
		}
		rows = append(rows, []string{s.Domain, s.Upstream, health})
	}
	o.Table(headers, rows)
	return nil
}

// saveConfig writes config back to YAML file
func saveConfig(cfg *config.Config) error {
	path := cfgFile
	if path == "" {
		found, err := config.FindConfigFile()
		if err != nil {
			path = "auto-ingress.yaml"
		} else {
			path = found
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	header := "# Auto Ingress Configuration\n# Managed by: auto-ingress CLI\n\n"
	if err := os.WriteFile(path, []byte(header+string(data)), 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
