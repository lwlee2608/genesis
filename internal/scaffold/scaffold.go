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
		return fmt.Errorf("pnpm not found on PATH; create the frontend manually with: cd services && pnpm create vite %s --no-interactive --template react-ts", webName)
	}

	c := exec.Command("pnpm", "create", "vite", webName, "--no-interactive", "--template", "react-ts")
	c.Dir = servicesDir
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("pnpm create vite failed: %w", err)
	}

	if err := pruneFrontend(filepath.Join(servicesDir, webName), cfg); err != nil {
		return fmt.Errorf("prune frontend assets: %w", err)
	}
	return nil
}

// frontendCruft is the demo content the vite react-ts template ships with.
// Pruning it keeps generated repos from committing throwaway assets — most
// notably the binary src/assets/hero.png — into git history forever.
var frontendCruft = []string{
	"public/favicon.svg",
	"public/icons.svg",
	"src/assets",
	"src/App.css",
}

const appTsx = `function App() {
  return <h1>{{.AppName}}</h1>
}

export default App
`

const indexCSS = `:root {
  font-family: system-ui, sans-serif;
  line-height: 1.5;
}

body {
  margin: 0;
}
`

const webReadme = "# {{.AppName}}-web\n\nFrontend for {{.AppName}}, built with Vite + React + TypeScript.\n\n```bash\npnpm install\npnpm dev\n```\n"

func pruneFrontend(webDir string, cfg Config) error {
	for _, rel := range frontendCruft {
		if err := os.RemoveAll(filepath.Join(webDir, rel)); err != nil {
			return err
		}
	}

	// Replace the files that referenced the deleted assets with minimal stubs.
	stubs := map[string]string{
		"src/App.tsx":   appTsx,
		"src/index.css": indexCSS,
		"README.md":     webReadme,
	}
	for rel, tmplStr := range stubs {
		if err := renderString(filepath.Join(webDir, rel), tmplStr, cfg); err != nil {
			return err
		}
	}

	return stripFaviconLink(filepath.Join(webDir, "index.html"))
}

// stripFaviconLink drops the <link rel="icon"> referencing the deleted favicon
// so the dev server doesn't 404 on it.
func stripFaviconLink(indexPath string) error {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := strings.Split(string(data), "\n")
	kept := lines[:0]
	for _, l := range lines {
		if strings.Contains(l, `rel="icon"`) {
			continue
		}
		kept = append(kept, l)
	}
	return os.WriteFile(indexPath, []byte(strings.Join(kept, "\n")), 0644)
}

func renderString(destPath, tmplStr string, data Config) error {
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
