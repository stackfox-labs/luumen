package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"

	"luumen/internal/resolver"
	"luumen/internal/workspace"
)

func TestAddKnownToolAlias(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	rokitPath := filepath.Join(repo, workspace.RokitConfigFile)
	writeFile(t, rokitPath, "[tools]\n")

	rokit := &fakeInstaller{}
	wally := &fakeInstaller{}

	err := executeAddCommand(addCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: repo, HasRokitConfig: true, RokitConfigPath: rokitPath}, nil
		},
		rokitInstaller: rokit,
		wallyInstaller: wally,
	}, "rojo")
	if err != nil {
		t.Fatalf("expected alias add success, got: %v", err)
	}

	if rokit.addCalls != 1 || rokit.calls != 0 || wally.calls != 0 {
		t.Fatalf("expected rokit add only, got rokit.add=%d rokit.install=%d wally.install=%d", rokit.addCalls, rokit.calls, wally.calls)
	}
	if rokit.lastTool != "rojo-rbx/rojo" {
		t.Fatalf("expected rokit add tool rojo-rbx/rojo, got %q", rokit.lastTool)
	}
	assertTomlTableContainsValue(t, rokitPath, "tools", "rojo-rbx/rojo")
}

func TestAddExplicitToolRef(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	rokitPath := filepath.Join(repo, workspace.RokitConfigFile)
	writeFile(t, rokitPath, "[tools]\n")

	err := executeAddCommand(addCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: repo, HasRokitConfig: true, RokitConfigPath: rokitPath}, nil
		},
		rokitInstaller: &fakeInstaller{},
		wallyInstaller: &fakeInstaller{},
	}, "tool:my-org/my-tool@1.2.3")
	if err != nil {
		t.Fatalf("expected explicit tool add success, got: %v", err)
	}
	assertTomlTableContainsValue(t, rokitPath, "tools", "my-org/my-tool@1.2.3")
}

func TestAddExplicitToolRefWithoutVersion(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	rokitPath := filepath.Join(repo, workspace.RokitConfigFile)
	writeFile(t, rokitPath, "[tools]\n")

	rokit := &fakeInstaller{}
	err := executeAddCommand(addCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: repo, HasRokitConfig: true, RokitConfigPath: rokitPath}, nil
		},
		rokitInstaller: rokit,
		wallyInstaller: &fakeInstaller{},
	}, "tool:my-org/my-tool")
	if err != nil {
		t.Fatalf("expected versionless explicit tool add success, got: %v", err)
	}
	if rokit.addCalls != 1 {
		t.Fatalf("expected one rokit add call, got %d", rokit.addCalls)
	}
	if rokit.lastTool != "my-org/my-tool" {
		t.Fatalf("expected rokit add tool my-org/my-tool, got %q", rokit.lastTool)
	}
	assertTomlTableContainsValue(t, rokitPath, "tools", "my-org/my-tool")
}

func TestAddExplicitPackageRef(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	wallyPath := filepath.Join(repo, workspace.WallyConfigFile)
	writeFile(t, wallyPath, "[package]\nname = \"luumen/test\"\nversion = \"0.1.0\"\nregistry = \"https://github.com/UpliftGames/wally-index\"\nrealm = \"shared\"\n\n[dependencies]\n")

	rokit := &fakeInstaller{}
	wally := &fakeInstaller{}

	err := executeAddCommand(addCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: repo, HasWallyConfig: true, WallyConfigPath: wallyPath}, nil
		},
		rokitInstaller: rokit,
		wallyInstaller: wally,
	}, "pkg:sleitnick/knit")
	if err != nil {
		t.Fatalf("expected explicit package add success, got: %v", err)
	}

	if rokit.calls != 0 || wally.calls != 1 {
		t.Fatalf("expected wally install only, got rokit=%d wally=%d", rokit.calls, wally.calls)
	}
	assertTomlTableContainsValue(t, wallyPath, "dependencies", "sleitnick/knit")
}

func TestAddAmbiguousValueErrors(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	rokitPath := filepath.Join(repo, workspace.RokitConfigFile)
	wallyPath := filepath.Join(repo, workspace.WallyConfigFile)
	writeFile(t, rokitPath, "[tools]\n")
	writeFile(t, wallyPath, "[dependencies]\n")

	rokit := &fakeInstaller{}
	wally := &fakeInstaller{}

	err := executeAddCommand(addCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{
				RootPath:        repo,
				HasRokitConfig:  true,
				HasWallyConfig:  true,
				RokitConfigPath: rokitPath,
				WallyConfigPath: wallyPath,
			}, nil
		},
		rokitInstaller: rokit,
		wallyInstaller: wally,
	}, "unknown")
	if err == nil {
		t.Fatal("expected ambiguous resolution error")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity error, got: %v", err)
	}
	if rokit.calls != 0 || rokit.addCalls != 0 || wally.calls != 0 {
		t.Fatalf("expected no installers on ambiguity, got rokit.install=%d rokit.add=%d wally.install=%d", rokit.calls, rokit.addCalls, wally.calls)
	}
}

func TestAddWithNoInstall(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	rokitPath := filepath.Join(repo, workspace.RokitConfigFile)
	writeFile(t, rokitPath, "[tools]\n")

	rokit := &fakeInstaller{}

	err := executeAddCommand(addCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: repo, HasRokitConfig: true, RokitConfigPath: rokitPath}, nil
		},
		rokitInstaller: rokit,
		wallyInstaller: &fakeInstaller{},
	}, "--no-install", "rojo")
	if err != nil {
		t.Fatalf("expected add --no-install success, got: %v", err)
	}
	if rokit.calls != 0 {
		t.Fatalf("expected install skip, got rokit=%d", rokit.calls)
	}
	if rokit.addCalls != 0 {
		t.Fatalf("expected rokit add skip, got %d", rokit.addCalls)
	}
	assertTomlTableContainsValue(t, rokitPath, "tools", "rojo-rbx/rojo@7.6.1")
}

func TestAddAvoidsDuplicateEntries(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	rokitPath := filepath.Join(repo, workspace.RokitConfigFile)
	writeFile(t, rokitPath, "[tools]\nrojo = \"rojo-rbx/rojo@7.6.1\"\n")

	err := executeAddCommand(addCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: repo, HasRokitConfig: true, RokitConfigPath: rokitPath}, nil
		},
		rokitInstaller: &fakeInstaller{},
		wallyInstaller: &fakeInstaller{},
	}, "--no-install", "tool:rojo-rbx/rojo")
	if err != nil {
		t.Fatalf("expected duplicate add success, got: %v", err)
	}

	contents, readErr := os.ReadFile(rokitPath)
	if readErr != nil {
		t.Fatalf("failed to read rokit config: %v", readErr)
	}
	var doc map[string]any
	if err := toml.Unmarshal(contents, &doc); err != nil {
		t.Fatalf("failed to decode rokit config: %v", err)
	}
	toolsTable, ok := doc["tools"].(map[string]any)
	if !ok {
		t.Fatalf("expected [tools] table, got: %#v", doc["tools"])
	}
	if len(toolsTable) != 1 {
		t.Fatalf("expected single rojo entry, got: %#v", toolsTable)
	}
	if _, ok := toolsTable["rojo"]; !ok {
		t.Fatalf("expected rojo key in tools table, got: %#v", toolsTable)
	}
}

func TestAddToolAlreadyDeclaredSkipsRokitAdd(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	rokitPath := filepath.Join(repo, workspace.RokitConfigFile)
	writeFile(t, rokitPath, "[tools]\nselene = \"Kampfkarren/selene@0.30.1\"\n")

	rokit := &fakeInstaller{}
	err := executeAddCommand(addCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: repo, HasRokitConfig: true, RokitConfigPath: rokitPath}, nil
		},
		rokitInstaller: rokit,
		wallyInstaller: &fakeInstaller{},
	}, "selene")
	if err != nil {
		t.Fatalf("expected idempotent add success, got: %v", err)
	}
	if rokit.addCalls != 0 {
		t.Fatalf("expected no rokit add call when already declared, got %d", rokit.addCalls)
	}
}

func TestAddToolUsesExecutableKey(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	rokitPath := filepath.Join(repo, workspace.RokitConfigFile)
	writeFile(t, rokitPath, "[tools]\n")

	err := executeAddCommand(addCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: repo, HasRokitConfig: true, RokitConfigPath: rokitPath}, nil
		},
		rokitInstaller: &fakeInstaller{},
		wallyInstaller: &fakeInstaller{},
	}, "--no-install", "tool:Kampfkarren/selene@0.30.1")
	if err != nil {
		t.Fatalf("expected add success, got: %v", err)
	}

	contents, readErr := os.ReadFile(rokitPath)
	if readErr != nil {
		t.Fatalf("failed to read rokit config: %v", readErr)
	}
	text := strings.ToLower(string(contents))
	if !strings.Contains(text, "selene =") {
		t.Fatalf("expected [tools].selene key, got: %s", string(contents))
	}
}

func TestRokitAddInvocationOmitsDefaultAlias(t *testing.T) {
	t.Parallel()

	tool, alias := rokitAddInvocation(resolver.Resolution{
		Kind:   resolver.DependencyKindTool,
		Source: "alias",
		Value:  "Kampfkarren/selene@0.30.1",
		Alias:  "selene",
	})

	if tool != "kampfkarren/selene" {
		t.Fatalf("expected canonical tool base, got %q", tool)
	}
	if alias != "" {
		t.Fatalf("expected default alias to be omitted, got %q", alias)
	}
}

func TestRokitAddInvocationKeepsCustomAlias(t *testing.T) {
	t.Parallel()

	tool, alias := rokitAddInvocation(resolver.Resolution{
		Kind:   resolver.DependencyKindTool,
		Source: "explicit",
		Value:  "my-org/my-tool",
		Alias:  "custom-tool",
	})

	if tool != "my-org/my-tool" {
		t.Fatalf("expected tool to remain unchanged, got %q", tool)
	}
	if alias != "custom-tool" {
		t.Fatalf("expected custom alias to be preserved, got %q", alias)
	}
}

func executeAddCommand(deps addCommandDeps, args ...string) error {
	cmd := newAddCmd(deps)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs(args)
	return cmd.Execute()
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func assertTomlTableContainsValue(t *testing.T, path string, table string, expected string) {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	var doc map[string]any
	if err := toml.Unmarshal(contents, &doc); err != nil {
		t.Fatalf("failed to decode %s: %v", path, err)
	}

	typedTable, ok := doc[table].(map[string]any)
	if !ok {
		t.Fatalf("expected [%s] table in %s", table, path)
	}

	for _, value := range typedTable {
		if stringValue, ok := value.(string); ok && stringValue == expected {
			return
		}
	}

	t.Fatalf("expected [%s] in %s to contain %q, got %#v", table, path, expected, typedTable)
}
