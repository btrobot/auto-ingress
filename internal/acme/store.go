package acme

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CertStatus represents certificate status
type CertStatus string

const (
	CertStatusValid   CertStatus = "valid"
	CertStatusExpired CertStatus = "expired"
	CertStatusPending CertStatus = "pending"
	CertStatusError   CertStatus = "error"
)

// CertInfo holds certificate metadata
type CertInfo struct {
	Domain    string     `json:"domain"`
	Status    CertStatus `json:"status"`
	Issuer    string     `json:"issuer"`
	NotBefore time.Time  `json:"not_before"`
	NotAfter  time.Time  `json:"not_after"`
	DaysLeft  int        `json:"days_left"`
	Error     string     `json:"error,omitempty"`
}

// Store reads certificate information from Caddy's data directory
type Store struct {
	dataDir string
}

// NewStore creates a certificate store
func NewStore(dataDir string) *Store {
	if dataDir == "" {
		dataDir = "/var/lib/docker/volumes/auto-ingress-data/_data"
	}
	return &Store{dataDir: dataDir}
}

// List returns all certificates found in the Caddy data directory
func (s *Store) List() ([]CertInfo, error) {
	certDir := filepath.Join(s.dataDir, "certificates")
	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		return nil, nil
	}

	var certs []CertInfo
	err := filepath.Walk(certDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(path, ".crt") {
			return nil
		}
		cert, err := parseCert(path)
		if err != nil {
			domain := extractDomainFromPath(path)
			certs = append(certs, CertInfo{
				Domain: domain,
				Status: CertStatusError,
				Error:  err.Error(),
			})
			return nil
		}
		info2 := certToInfo(cert, extractDomainFromPath(path))
		certs = append(certs, info2)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk certificates: %w", err)
	}
	return certs, nil
}

// Status returns certificate info for a specific domain
func (s *Store) Status(domain string) (*CertInfo, error) {
	certs, err := s.List()
	if err != nil {
		return nil, err
	}
	for _, c := range certs {
		if c.Domain == domain {
			return &c, nil
		}
	}
	return nil, nil
}

func parseCert(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM data found")
	}
	return x509.ParseCertificate(block.Bytes)
}

func certToInfo(cert *x509.Certificate, domain string) CertInfo {
	now := time.Now()
	daysLeft := int(time.Until(cert.NotAfter).Hours() / 24)

	status := CertStatusValid
	if now.After(cert.NotAfter) {
		status = CertStatusExpired
		daysLeft = -daysLeft
	} else if daysLeft < 30 {
		status = CertStatusValid // could add CertStatusExpiringSoon
	}

	return CertInfo{
		Domain:    domain,
		Status:    status,
		Issuer:    cert.Issuer.CommonName,
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
		DaysLeft:  daysLeft,
	}
}

func extractDomainFromPath(path string) string {
	// Path is like: .../certificates/acme-v02.api.letsencrypt.org-directory/example.com/example.com.crt
	dir := filepath.Dir(path)
	return filepath.Base(dir)
}
