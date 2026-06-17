package scaffold

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
		if err := writeTemplate(filepath.Join(webDir, rel), tmplStr, cfg); err != nil {
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
