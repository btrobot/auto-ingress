package caddy

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/nodecoda/auto-ingress/internal/config"
)

//go:embed templates/Caddyfile.gotmpl
var templateFS embed.FS

// Generator generates Caddyfile from config
type Generator struct{}

// NewGenerator creates a new Caddyfile generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate produces a Caddyfile from the given config
func (g *Generator) Generate(cfg *config.Config) ([]byte, error) {
	tmplData, err := templateFS.ReadFile("templates/Caddyfile.gotmpl")
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}

	tmpl, err := template.New("caddyfile").Parse(string(tmplData))
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateToFile writes the Caddyfile to a path
func (g *Generator) GenerateToFile(cfg *config.Config, path string) error {
	data, err := g.Generate(cfg)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// Preview returns a formatted preview of the generated Caddyfile
func (g *Generator) Preview(cfg *config.Config) (string, error) {
	data, err := g.Generate(cfg)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("┌" + strings.Repeat("─", 62) + "┐\n")
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		sb.WriteString(fmt.Sprintf("│ %-60s │\n", line))
	}
	sb.WriteString("└" + strings.Repeat("─", 62) + "┘\n")
	return sb.String(), nil
}
