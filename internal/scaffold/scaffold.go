package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

const MarkerFile = ".genesis.json"

type Config struct {
	AppName    string
	ModuleName string
	AddHTTP    bool
	FullStack  bool
	OutputDir  string
	Version    string
}

type marker struct {
	Version   string `json:"version"`
	Created   string `json:"created"`
	FullStack bool   `json:"fullstack"`
	HTTP      bool   `json:"http"`
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

	return writeMarker(cfg)
}

func writeMarker(cfg Config) error {
	data, err := json.MarshalIndent(marker{
		Version:   cfg.Version,
		Created:   time.Now().Format(time.DateOnly),
		FullStack: cfg.FullStack,
		HTTP:      cfg.AddHTTP,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cfg.OutputDir, MarkerFile), append(data, '\n'), 0644)
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
