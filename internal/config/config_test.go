package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
[project]
name = "my-game"

[install]
tools = true
packages = true

[commands]
serve = "rojo serve"
dev = ["luu sourcemap", "rojo serve"]

[tasks]
fmt = "stylua src"
ci = ["luu install", "luu build"]
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}

	if cfg.Project.Name != "my-game" {
		t.Fatalf("expected project name my-game, got %q", cfg.Project.Name)
	}

	serve := cfg.Commands["serve"].Commands
	if len(serve) != 1 || serve[0] != "rojo serve" {
		t.Fatalf("expected serve to normalize to one command, got %#v", serve)
	}

	dev := cfg.Commands["dev"].Commands
	if len(dev) != 2 || dev[0] != "luu sourcemap" || dev[1] != "rojo serve" {
		t.Fatalf("expected dev commands to remain ordered, got %#v", dev)
	}

	ci := cfg.Tasks["ci"].Commands
	if len(ci) != 2 || ci[0] != "luu install" || ci[1] != "luu build" {
		t.Fatalf("expected ci task commands to remain ordered, got %#v", ci)
	}
}

func TestLoadInvalidConfigValue(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
[tasks]
bad = 42
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected an error for invalid task value")
	}

	message := err.Error()
	if !strings.Contains(message, "tasks.bad") {
		t.Fatalf("expected error to mention tasks.bad, got %q", message)
	}
	if !strings.Contains(message, "string or array of strings") {
		t.Fatalf("expected actionable type message, got %q", message)
	}
}

func TestLoadMissingConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), FileName)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected missing config error")
	}
	if !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("expected ErrConfigNotFound, got %v", err)
	}
}

func TestWriteRoundTrip(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
[project]
name = "round-trip"

[tasks]
fmt = "stylua src"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}

	cfg.Tasks["lint"] = NewTaskValue("selene src")
	cfg.Commands = map[string]TaskValue{
		"build": NewTaskValue("rojo build -o build.rbxl"),
	}

	if err := Write(path, cfg); err != nil {
		t.Fatalf("expected write to succeed, got: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("expected reloaded config, got: %v", err)
	}

	if reloaded.Tasks["lint"].Commands[0] != "selene src" {
		t.Fatalf("expected lint task to persist, got %#v", reloaded.Tasks["lint"].Commands)
	}
	if reloaded.Commands["build"].Commands[0] != "rojo build -o build.rbxl" {
		t.Fatalf("expected build command to persist, got %#v", reloaded.Commands["build"].Commands)
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	return path
}
