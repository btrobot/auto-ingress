package cmd

import (
	"fmt"

	"github.com/nodecoda/auto-ingress/internal/acme"
	"github.com/nodecoda/auto-ingress/pkg/output"
	"github.com/spf13/cobra"
)

var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "Manage certificates",
	Long:  "View and manage Let's Encrypt certificates.",
}

var certListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all certificates",
	RunE:  runCertList,
}

var certStatusCmd = &cobra.Command{
	Use:   "status <domain>",
	Short: "Show certificate status for a domain",
	Args:  cobra.ExactArgs(1),
	RunE:  runCertStatus,
}

func init() {
	certCmd.AddCommand(certListCmd)
	certCmd.AddCommand(certStatusCmd)
}

func runCertList(cmd *cobra.Command, args []string) error {
	o := out()

	store := acme.NewStore("")
	certs, err := store.List()
	if err != nil {
		return fmt.Errorf("list certificates: %w", err)
	}

	if len(certs) == 0 {
		o.Info("No certificates found.")
		fmt.Println("  Certificates are created automatically when the gateway starts.")
		return nil
	}

	if jsonOutput {
		return o.JSON(certs)
	}

	headers := []string{"DOMAIN", "STATUS", "EXPIRES", "DAYS LEFT"}
	var rows [][]string
	for _, c := range certs {
		days := fmt.Sprintf("%d", c.DaysLeft)
		if c.DaysLeft < 30 {
			days = fmt.Sprintf("%d ⚠", c.DaysLeft)
		}
		expires := c.NotAfter.Format("2006-01-02")
		if c.NotAfter.IsZero() {
			expires = "—"
			days = "—"
		}
		rows = append(rows, []string{
			c.Domain,
			output.StatusIndicator(string(c.Status)),
			expires,
			days,
		})
	}
	o.Table(headers, rows)
	return nil
}

func runCertStatus(cmd *cobra.Command, args []string) error {
	o := out()
	domain := args[0]

	store := acme.NewStore("")
	cert, err := store.Status(domain)
	if err != nil {
		return fmt.Errorf("cert status: %w", err)
	}

	if cert == nil {
		o.Warning("No certificate found for %s", domain)
		return nil
	}

	if jsonOutput {
		return o.JSON(cert)
	}

	o.Header(fmt.Sprintf("Certificate: %s", domain))
	fmt.Printf("  Status:    %s %s\n", output.StatusIndicator(string(cert.Status)), cert.Status)
	fmt.Printf("  Issuer:    %s\n", cert.Issuer)
	fmt.Printf("  Valid:     %s → %s\n", cert.NotBefore.Format("2006-01-02"), cert.NotAfter.Format("2006-01-02"))
	fmt.Printf("  Days left: %d\n", cert.DaysLeft)
	if cert.Error != "" {
		fmt.Printf("  Error:     %s\n", cert.Error)
	}
	return nil
}
