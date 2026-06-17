package scaffold

import (
	"embed"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed all:templates
var templates embed.FS

func generateBackend(backendDir string, cfg Config) error {
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
	return tidyCmd.Run()
}

func generateFile(tmplPath, destPath string, data Config) error {
	content, err := templates.ReadFile(tmplPath)
	if err != nil {
		return err
	}

	return writeTemplate(destPath, string(content), data)
}
