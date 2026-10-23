package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/lwlee2608/go-bootstrap/internal/tui"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		wantNonInteractive bool
		wantHTTP           bool
		wantFullStack      bool
	}{
		{"no flags", nil, false, true, true},
		{"version only", []string{"-version"}, false, true, true},
		{"yes", []string{"-y"}, true, true, true},
		{"name", []string{"-name", "foo"}, true, true, true},
		{"fullstack off", []string{"-fullstack=false"}, true, true, false},
		{"http off", []string{"-http=false"}, true, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, nonInteractive, err := parseFlags(flag.NewFlagSet("genesis", flag.ContinueOnError), tt.args)
			if err != nil {
				t.Fatalf("parseFlags: %v", err)
			}
			if nonInteractive != tt.wantNonInteractive {
				t.Errorf("nonInteractive = %v, want %v", nonInteractive, tt.wantNonInteractive)
			}
			if o.http != tt.wantHTTP || o.fullStack != tt.wantFullStack {
				t.Errorf("http, fullStack = %v, %v, want %v, %v", o.http, o.fullStack, tt.wantHTTP, tt.wantFullStack)
			}
		})
	}
}

func TestFlagResult(t *testing.T) {
	tests := []struct {
		name    string
		opts    options
		apiKey  string
		want    tui.Result
		wantErr bool
	}{
		{
			name: "detected defaults",
			opts: options{http: true, fullStack: true},
			want: tui.Result{AppName: "detected", ModuleName: "github.com/u/detected", AddHTTP: true, FullStack: true},
		},
		{
			name: "flags override detection",
			opts: options{name: "foo", module: "github.com/u/foo"},
			want: tui.Result{AppName: "foo", ModuleName: "github.com/u/foo"},
		},
		{
			name:   "readme with key",
			opts:   options{readme: true, desc: "does things"},
			apiKey: "sk",
			want:   tui.Result{AppName: "detected", ModuleName: "github.com/u/detected", GenReadme: true, Description: "does things", APIKey: "sk"},
		},
		{
			name:    "readme without key",
			opts:    options{readme: true},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := flagResult(tt.opts, "detected", "github.com/u/detected", tt.apiKey)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFlagResultMissingDetection(t *testing.T) {
	if _, err := flagResult(options{}, "", "github.com/u/foo", ""); err == nil {
		t.Error("expected error when app name is missing")
	}
	if _, err := flagResult(options{}, "foo", "", ""); err == nil {
		t.Error("expected error when module is missing")
	}
}

func TestCheckEmptyDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := checkEmptyDir(dir); err != nil {
		t.Errorf("dir with only .git: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "README.md"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := checkEmptyDir(dir); err == nil {
		t.Error("expected error for non-empty dir")
	}
}
