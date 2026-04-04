package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectOnlyRokit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, RokitConfigFile))

	state, err := Detect(root)
	if err != nil {
		t.Fatalf("expected detection success, got: %v", err)
	}

	if !state.HasRokitConfig {
		t.Fatal("expected Rokit config to be detected")
	}
	if state.HasWallyConfig || state.HasRojoProject || state.HasLuumenConfig {
		t.Fatal("expected only Rokit config in this test")
	}
	if !state.IsAdoptable || state.IsLuumenManaged {
		t.Fatal("expected repo to be adoptable and not Luumen-managed")
	}
}

func TestDetectOnlyWally(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, WallyConfigFile))

	state, err := Detect(root)
	if err != nil {
		t.Fatalf("expected detection success, got: %v", err)
	}

	if !state.HasWallyConfig {
		t.Fatal("expected Wally config to be detected")
	}
	if state.HasRokitConfig || state.HasRojoProject || state.HasLuumenConfig {
		t.Fatal("expected only Wally config in this test")
	}
	if !state.IsAdoptable || state.IsLuumenManaged {
		t.Fatal("expected repo to be adoptable and not Luumen-managed")
	}
}

func TestDetectRokitWallyAndRojo(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, RokitConfigFile))
	mustWriteFile(t, filepath.Join(root, WallyConfigFile))
	mustWriteFile(t, filepath.Join(root, "default.project.json"))
	mustWriteFile(t, filepath.Join(root, "game.project.json"))

	state, err := Detect(root)
	if err != nil {
		t.Fatalf("expected detection success, got: %v", err)
	}

	if !state.HasRokitConfig || !state.HasWallyConfig || !state.HasRojoProject {
		t.Fatalf("expected Rokit, Wally, and Rojo files detected, got %#v", state)
	}
	if len(state.RojoProjectPaths) != 2 {
		t.Fatalf("expected two Rojo project files, got %#v", state.RojoProjectPaths)
	}
	if !state.IsAdoptable || state.IsLuumenManaged {
		t.Fatal("expected repo to be adoptable and not Luumen-managed")
	}
}

func TestDetectLuumenManaged(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, LuumenConfigFile))
	mustWriteFile(t, filepath.Join(root, RokitConfigFile))

	state, err := Detect(root)
	if err != nil {
		t.Fatalf("expected detection success, got: %v", err)
	}

	if !state.IsLuumenManaged {
		t.Fatal("expected workspace to be Luumen-managed")
	}
	if state.IsAdoptable {
		t.Fatal("expected managed workspace to not be marked as adoptable")
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()

	if err := os.WriteFile(path, []byte("placeholder\n"), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
