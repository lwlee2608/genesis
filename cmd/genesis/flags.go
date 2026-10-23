package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lwlee2608/go-bootstrap/internal/tui"
	"github.com/mattn/go-isatty"
)

type options struct {
	version   bool
	yes       bool
	force     bool
	name      string
	module    string
	http      bool
	fullStack bool
	readme    bool
	desc      string
}

func parseFlags(fs *flag.FlagSet, args []string) (options, bool, error) {
	var o options
	fs.BoolVar(&o.version, "version", false, "print version")
	fs.BoolVar(&o.yes, "y", false, "non-interactive: accept detected defaults")
	fs.BoolVar(&o.force, "force", false, "scaffold into a non-empty directory")
	fs.StringVar(&o.name, "name", "", "app name (default: current directory name)")
	fs.StringVar(&o.module, "module", "", "Go module path (default: from git remote origin)")
	fs.BoolVar(&o.http, "http", true, "add HTTP scaffolding")
	fs.BoolVar(&o.fullStack, "fullstack", true, "nest backend under services/<name>-server and add services/<name>-web")
	fs.BoolVar(&o.readme, "readme", false, "generate AI README (needs OPENROUTER_API_KEY)")
	fs.StringVar(&o.desc, "desc", "", "short app description for the AI README")
	if err := fs.Parse(args); err != nil {
		return options{}, false, err
	}

	nonInteractive := o.yes
	fs.Visit(func(f *flag.Flag) {
		if f.Name != "version" {
			nonInteractive = true
		}
	})
	return o, nonInteractive, nil
}

func flagResult(o options, detectedApp, detectedModule, apiKey string) (tui.Result, error) {
	r := tui.Result{
		AppName:     o.name,
		ModuleName:  o.module,
		AddHTTP:     o.http,
		FullStack:   o.fullStack,
		GenReadme:   o.readme,
		Description: o.desc,
	}
	if r.AppName == "" {
		r.AppName = detectedApp
	}
	if r.ModuleName == "" {
		r.ModuleName = detectedModule
	}
	if r.AppName == "" {
		return tui.Result{}, fmt.Errorf("app name not detected; pass -name")
	}
	if r.ModuleName == "" {
		return tui.Result{}, fmt.Errorf("module not detected from git remote; pass -module")
	}
	if r.GenReadme {
		if apiKey == "" {
			return tui.Result{}, fmt.Errorf("-readme needs OPENROUTER_API_KEY")
		}
		r.APIKey = apiKey
	}
	return r, nil
}

func checkEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() != ".git" {
			return fmt.Errorf("%s is not empty (found %s); pass -force to scaffold anyway", dir, e.Name())
		}
	}
	return nil
}

func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd())
}
