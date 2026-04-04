package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"

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

	if rokit.calls != 1 || wally.calls != 0 {
		t.Fatalf("expected rokit install only, got rokit=%d wally=%d", rokit.calls, wally.calls)
	}
	assertTomlTableContainsValue(t, rokitPath, "tools", "rojo-rbx/rojo@7.6.1")
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
	if rokit.calls != 0 || wally.calls != 0 {
		t.Fatalf("expected no installers on ambiguity, got rokit=%d wally=%d", rokit.calls, wally.calls)
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
	}, "tool:rojo-rbx/rojo")
	if err != nil {
		t.Fatalf("expected duplicate add success, got: %v", err)
	}

	contents, readErr := os.ReadFile(rokitPath)
	if readErr != nil {
		t.Fatalf("failed to read rokit config: %v", readErr)
	}
	if strings.Count(string(contents), "rojo-rbx/rojo@7.6.1") != 1 {
		t.Fatalf("expected one canonical tool entry, got: %s", string(contents))
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
