package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lwlee2608/go-bootstrap/internal/git"
	"github.com/lwlee2608/go-bootstrap/internal/readme"
	"github.com/lwlee2608/go-bootstrap/internal/scaffold"
	"github.com/lwlee2608/go-bootstrap/internal/tui"
	"github.com/lwlee2608/go-bootstrap/internal/update"
)

var AppVersion = "dev"

func main() {
	opts, nonInteractive, err := parseFlags(flag.CommandLine, os.Args[1:])
	if err != nil {
		os.Exit(2)
	}

	notice, waitForUpdateCheck := update.Check(AppVersion)
	defer waitForUpdateCheck()

	if opts.version {
		fmt.Printf("genesis %s\n", AppVersion)
		if notice != "" {
			fmt.Println("\n" + notice)
		}
		return
	}

	if notice != "" {
		fmt.Println(notice + "\n")
	}

	outputDir := "."

	suggestedApp := git.DetectAppName()
	suggestedModule := git.DetectModuleName()

	var result tui.Result
	if nonInteractive {
		result, err = flagResult(opts, suggestedApp, suggestedModule, readme.EnvKey())
		if err == nil && !opts.force {
			err = checkEmptyDir(outputDir)
		}
		if err == nil && result.GenReadme {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			err = readme.ValidateKey(ctx, result.APIKey)
			cancel()
		}
	} else {
		if !isTerminal(os.Stdin) {
			fmt.Fprintln(os.Stderr, "Error: no TTY for interactive prompts; pass flags (or -y) to run non-interactively")
			flag.PrintDefaults()
			os.Exit(1)
		}
		result, err = runTUI(suggestedApp, suggestedModule)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg := scaffold.Config{
		AppName:    result.AppName,
		ModuleName: result.ModuleName,
		AddHTTP:    result.AddHTTP,
		FullStack:  result.FullStack,
		OutputDir:  outputDir,
	}

	if err := scaffold.Generate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating project: %v\n", err)
		os.Exit(1)
	}

	if result.GenReadme {
		fmt.Println("\nGenerating README.md with AI...")
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if err := readme.Generate(ctx, readme.Config{
			AppName:     result.AppName,
			ModuleName:  result.ModuleName,
			Description: result.Description,
			FullStack:   result.FullStack,
			OutputDir:   outputDir,
			APIKey:      result.APIKey,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping AI README: %v\n", err)
		}
	}

	fmt.Println("\nProject created!")
	fmt.Println("\nNext steps:")
	if result.FullStack {
		fmt.Printf("  cd services/%s-server && make build\n", result.AppName)
		fmt.Printf("  cd services/%s-web && pnpm install\n", result.AppName)
	} else {
		fmt.Println("  make build")
		fmt.Printf("  ./bin/%s\n", result.AppName)
	}
}

func runTUI(suggestedApp, suggestedModule string) (tui.Result, error) {
	model := tui.New(tui.Options{
		SuggestedAppName:    suggestedApp,
		SuggestedModuleName: suggestedModule,
	})
	finalModel, err := tea.NewProgram(model).Run()
	if err != nil {
		return tui.Result{}, err
	}
	return finalModel.(tui.Model).Result()
}
