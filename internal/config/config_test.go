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
return {
    project = {
        name = "my-game",
        version = "0.1.0",
        author = "Omouta",
        description = "Example project",
    },

    install = {
        tools = true,
        packages = true,
    },

    tools = {
        rojo = "rojo-rbx/rojo@7.6.1",
    },

    packages = {
        roact = "roblox/roact@1.4.4",
    },

    commands = {
        serve = "rojo serve",
        dev = {
            "luu sourcemap",
            "rojo serve",
        },
    },

    tasks = {
        fmt = "stylua src",
        ci = {
            "luu install",
            "luu build",
        },
    },
}
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}

	if cfg.Project.Name != "my-game" || cfg.Project.Version != "0.1.0" || cfg.Project.Author != "Omouta" || cfg.Project.Description != "Example project" {
		t.Fatalf("expected project metadata to load, got %+v", cfg.Project)
	}
	if !cfg.Install.Tools || !cfg.Install.Packages {
		t.Fatalf("expected install flags to load, got %+v", cfg.Install)
	}
	if cfg.Tools["rojo"] != "rojo-rbx/rojo@7.6.1" {
		t.Fatalf("expected tools.rojo to load, got %+v", cfg.Tools)
	}
	if cfg.Packages["roact"] != "roblox/roact@1.4.4" {
		t.Fatalf("expected packages.roact to load, got %+v", cfg.Packages)
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

func TestLoadInvalidSyntax(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
return {
    tasks = {
        fmt = "stylua src",
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected syntax error")
	}

	if !strings.Contains(err.Error(), "invalid Luau syntax") {
		t.Fatalf("expected invalid Luau syntax error, got %q", err.Error())
	}
}

func TestLoadUnsupportedConstruct(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
return {
    tasks = {
        fmt = string.format("stylua %s", "src"),
    },
}
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected unsupported construct error")
	}

	if !strings.Contains(err.Error(), "function calls are not allowed") {
		t.Fatalf("expected unsupported construct message, got %q", err.Error())
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

func TestLoadCommandAsString(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
return {
    commands = {
        build = "rojo build default.project.json --output build.rbxl",
    },
}
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected config to load, got: %v", err)
	}

	if got := cfg.Commands["build"].Commands; len(got) != 1 || got[0] != "rojo build default.project.json --output build.rbxl" {
		t.Fatalf("expected single build command, got %#v", got)
	}
}

func TestLoadCommandAsArray(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
return {
    commands = {
        dev = {
            "rojo sourcemap default.project.json --output sourcemap.json",
            "rojo serve default.project.json",
        },
    },
}
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected config to load, got: %v", err)
	}

	got := cfg.Commands["dev"].Commands
	expected := []string{
		"rojo sourcemap default.project.json --output sourcemap.json",
		"rojo serve default.project.json",
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %#v, got %#v", expected, got)
	}
	for index := range expected {
		if got[index] != expected[index] {
			t.Fatalf("expected %#v, got %#v", expected, got)
		}
	}
}

func TestLoadTaskAsString(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
return {
    tasks = {
        fmt = "stylua src",
    },
}
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected config to load, got: %v", err)
	}

	if got := cfg.Tasks["fmt"].Commands; len(got) != 1 || got[0] != "stylua src" {
		t.Fatalf("expected single fmt task, got %#v", got)
	}
}

func TestLoadTaskAsArray(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
return {
    tasks = {
        ci = {
            "luu install",
            "luu build",
        },
    },
}
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected config to load, got: %v", err)
	}

	got := cfg.Tasks["ci"].Commands
	expected := []string{"luu install", "luu build"}
	if len(got) != len(expected) {
		t.Fatalf("expected %#v, got %#v", expected, got)
	}
	for index := range expected {
		if got[index] != expected[index] {
			t.Fatalf("expected %#v, got %#v", expected, got)
		}
	}
}

func TestWriteRoundTrip(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
return {
    project = {
        name = "round-trip",
    },

    tasks = {
        fmt = "stylua src",
    },
}
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected config to load, got error: %v", err)
	}

	cfg.Project.Description = "Round trip test"
	cfg.Tools = map[string]string{
		"rojo": "rojo-rbx/rojo@7.6.1",
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

	if reloaded.Project.Description != "Round trip test" {
		t.Fatalf("expected description to persist, got %+v", reloaded.Project)
	}
	if reloaded.Tools["rojo"] != "rojo-rbx/rojo@7.6.1" {
		t.Fatalf("expected tools to persist, got %+v", reloaded.Tools)
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
