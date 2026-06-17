package scaffold

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:templates
var templates embed.FS

type Config struct {
	AppName    string
	ModuleName string
	AddHTTP    bool
	FullStack  bool
	OutputDir  string
}

func Generate(cfg Config) error {
	backendDir := cfg.OutputDir
	if cfg.FullStack {
		backendDir = filepath.Join(cfg.OutputDir, "services", cfg.AppName+"-server")
	}

	// Create backend directory if it doesn't exist
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		return err
	}

	// Create pkg directory explicitly
	if err := os.MkdirAll(filepath.Join(backendDir, "pkg"), 0755); err != nil {
		return err
	}

	err := fs.WalkDir(templates, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Skip files not ending in .tmpl (if any)
		if !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Calculate relative path from "templates" directory
		relPath, err := filepath.Rel("templates", path)
		if err != nil {
			return err
		}

		// Special handling for HTTP files if AddHTTP is false
		if !cfg.AddHTTP {
			if strings.HasPrefix(relPath, "internal/api/http") ||
				relPath == "cmd/config.go.tmpl" ||
				relPath == "cmd/logger.go.tmpl" ||
				relPath == "application.yml.tmpl" {
				return nil
			}
		}

		// Determine destination path
		destRelPath := strings.TrimSuffix(relPath, ".tmpl")
		if destRelPath == "gitignore" {
			destRelPath = ".gitignore"
		}
		if strings.HasPrefix(destRelPath, "cmd/") {
			// Map cmd/main.go -> cmd/{appName}/main.go
			destRelPath = filepath.Join("cmd", cfg.AppName, strings.TrimPrefix(destRelPath, "cmd/"))
		}

		destPath := filepath.Join(backendDir, destRelPath)

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		return generateFile(path, destPath, cfg)
	})

	if err != nil {
		return err
	}

	// Run go mod tidy at the backend root
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = backendDir
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	if err := tidyCmd.Run(); err != nil {
		return err
	}

	if cfg.FullStack {
		if err := createFrontend(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	return nil
}

func createFrontend(cfg Config) error {
	servicesDir := filepath.Join(cfg.OutputDir, "services")
	if err := os.MkdirAll(servicesDir, 0755); err != nil {
		return err
	}

	webName := cfg.AppName + "-web"
	if _, err := exec.LookPath("pnpm"); err != nil {
		return fmt.Errorf("pnpm not found on PATH; create the frontend manually with: cd services && pnpm create vite %s --template react-ts", webName)
	}

	c := exec.Command("pnpm", "create", "vite", webName, "--template", "react-ts")
	c.Dir = servicesDir
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("pnpm create vite failed: %w", err)
	}
	return nil
}

func generateFile(tmplPath, destPath string, data Config) error {
	content, err := templates.ReadFile(tmplPath)
	if err != nil {
		return err
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(content))
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
