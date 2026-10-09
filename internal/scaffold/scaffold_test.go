package scaffold

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		AppName:    "testapp",
		ModuleName: "github.com/test/testapp",
		OutputDir:  tmpDir,
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check directories exist
	dirs := []string{
		filepath.Join(tmpDir, "cmd", "testapp"),
		filepath.Join(tmpDir, "pkg"),
	}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("directory not created: %s", dir)
		}
	}

	// Check files exist and have correct content
	gomod, err := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}
	if !strings.Contains(string(gomod), "github.com/test/testapp") {
		t.Error("go.mod does not contain module name")
	}

	makefile, err := os.ReadFile(filepath.Join(tmpDir, "Makefile"))
	if err != nil {
		t.Fatalf("failed to read Makefile: %v", err)
	}
	if !strings.Contains(string(makefile), "APP             := testapp") {
		t.Error("Makefile does not contain app name")
	}

	mainGo, err := os.ReadFile(filepath.Join(tmpDir, "cmd", "testapp", "main.go"))
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}
	if !strings.Contains(string(mainGo), "testapp") {
		t.Error("main.go does not contain app name")
	}

	gitignore, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}
	if !strings.Contains(string(gitignore), "bin/") {
		t.Error(".gitignore does not contain bin/")
	}
}

const viteReactPackageJSON = `{
  "name": "myapp-web",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "lint": "eslint .",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^19.2.6",
    "react-dom": "^19.2.6"
  },
  "devDependencies": {
    "@eslint/js": "^10.0.1",
    "@types/node": "^24.12.3",
    "@vitejs/plugin-react": "^6.0.1",
    "eslint": "^10.3.0",
    "eslint-plugin-react-hooks": "^7.1.1",
    "eslint-plugin-react-refresh": "^0.5.2",
    "globals": "^17.6.0",
    "typescript": "~6.0.2",
    "typescript-eslint": "^8.59.2",
    "vite": "^8.0.12"
  }
}
`

func TestConfigureFrontendLint(t *testing.T) {
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(viteReactPackageJSON)); err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	if err := json.Unmarshal([]byte(viteReactPackageJSON), &want); err != nil {
		t.Fatal(err)
	}
	want["scripts"].(map[string]any)["lint"] = "oxlint"
	want["devDependencies"] = map[string]any{
		"@types/node": "^24.12.3", "@vitejs/plugin-react": "^6.0.1",
		"oxlint": oxlintVersion, "typescript": "~6.0.2", "vite": "^8.0.12",
	}
	for name, input := range map[string]string{
		"pretty":         viteReactPackageJSON,
		"compact":        compact.String(),
		"CRLF":           strings.ReplaceAll(viteReactPackageJSON, "\n", "\r\n"),
		"inline scripts": strings.Replace(viteReactPackageJSON, `"scripts": {`+"\n", `"scripts": { `, 1),
	} {
		t.Run(name, func(t *testing.T) {
			webDir := t.TempDir()
			path := filepath.Join(webDir, "package.json")
			if err := os.WriteFile(path, []byte(input), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(webDir, "eslint.config.js"), []byte("old config"), 0644); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				if err := configureFrontendLint(webDir); err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var got map[string]any
				if err := json.Unmarshal(data, &got); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("package.json mismatch:\n%s", data)
				}
			}
			if _, err := os.Stat(filepath.Join(webDir, "eslint.config.js")); !os.IsNotExist(err) {
				t.Fatal("ESLint config was not removed")
			}
			config, err := os.ReadFile(filepath.Join(webDir, ".oxlintrc.json"))
			if err != nil {
				t.Fatal(err)
			}
			if string(config) != oxlintConfig {
				t.Fatal("unexpected Oxlint config")
			}
		})
	}
}

func TestFrontendLintMatchesReference(t *testing.T) {
	webDir := filepath.Join("..", "..", "reference", "project-00", "services", "project-00-web")
	config, err := os.ReadFile(filepath.Join(webDir, ".oxlintrc.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(config) != oxlintConfig {
		t.Fatal("CLI and reference Oxlint configs differ")
	}
	data, err := os.ReadFile(filepath.Join(webDir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct {
		Scripts         map[string]string
		DevDependencies map[string]string
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.Scripts["lint"] != "oxlint" || pkg.DevDependencies["oxlint"] != oxlintVersion {
		t.Fatal("CLI and reference Oxlint script or version differ")
	}
}

func TestPruneFrontend(t *testing.T) {
	webDir := t.TempDir()

	// Simulate the demo content vite's react-ts template emits.
	seed := map[string]string{
		"public/favicon.svg":   "<svg/>",
		"public/icons.svg":     "<svg/>",
		"src/assets/hero.png":  "binary",
		"src/assets/react.svg": "<svg/>",
		"src/App.css":          ".demo{}",
		"src/App.tsx":          "import hero from './assets/hero.png'",
		"src/index.css":        ":root{--demo:1}",
		"README.md":            "# Vite boilerplate",
		"index.html":           "<head>\n  <link rel=\"icon\" href=\"/favicon.svg\" />\n  <title>x</title>\n</head>",
	}
	for rel, content := range seed {
		path := filepath.Join(webDir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := pruneFrontend(webDir, Config{AppName: "myapp"}); err != nil {
		t.Fatalf("pruneFrontend failed: %v", err)
	}

	// Demo assets must be gone so they never enter git history.
	gone := []string{"public/favicon.svg", "public/icons.svg", "src/assets", "src/App.css"}
	for _, rel := range gone {
		if _, err := os.Stat(filepath.Join(webDir, rel)); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed", rel)
		}
	}

	appTsx, err := os.ReadFile(filepath.Join(webDir, "src", "App.tsx"))
	if err != nil {
		t.Fatalf("failed to read App.tsx: %v", err)
	}
	if strings.Contains(string(appTsx), "assets") {
		t.Error("App.tsx still references deleted assets")
	}
	if !strings.Contains(string(appTsx), "myapp") {
		t.Error("App.tsx does not contain app name")
	}

	indexHTML, err := os.ReadFile(filepath.Join(webDir, "index.html"))
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}
	if strings.Contains(string(indexHTML), `rel="icon"`) {
		t.Error("index.html still references deleted favicon")
	}
}
