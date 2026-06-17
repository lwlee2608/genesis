package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

type Config struct {
	AppName    string
	ModuleName string
	AddHTTP    bool
	FullStack  bool
	OutputDir  string
}

// Generate scaffolds the backend and, for full-stack projects, the frontend.
func Generate(cfg Config) error {
	backendDir := cfg.OutputDir
	if cfg.FullStack {
		backendDir = filepath.Join(cfg.OutputDir, "services", cfg.AppName+"-server")
	}

	if err := generateBackend(backendDir, cfg); err != nil {
		return err
	}

	if cfg.FullStack {
		if err := createFrontend(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	return nil
}

// writeTemplate renders tmplStr with data and writes the result to destPath.
func writeTemplate(destPath, tmplStr string, data Config) error {
	tmpl, err := template.New(filepath.Base(destPath)).Parse(tmplStr)
	if err != nil {
		return err
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}
