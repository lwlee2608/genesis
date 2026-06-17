package scaffold

import (
	"os"
	"path/filepath"
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
