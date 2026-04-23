package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	LuumenConfigFile = ".config.luau"
	RokitConfigFile  = "rokit.toml"
	WallyConfigFile  = "wally.toml"
)

type Workspace struct {
	RootPath string

	LuumenConfigPath string
	RokitConfigPath  string
	WallyConfigPath  string
	RojoProjectPaths []string

	HasLuumenConfig bool
	HasRokitConfig  bool
	HasWallyConfig  bool
	HasRojoProject  bool

	IsLuumenManaged bool
	IsAdoptable     bool
}

func Detect(rootPath string) (Workspace, error) {
	if rootPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return Workspace{}, fmt.Errorf("failed to resolve current directory: %w", err)
		}
		rootPath = cwd
	}

	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return Workspace{}, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return Workspace{}, fmt.Errorf("failed to inspect workspace root: %w", err)
	}
	if !info.IsDir() {
		return Workspace{}, fmt.Errorf("workspace root is not a directory: %s", absRoot)
	}

	state := Workspace{RootPath: absRoot}

	state.LuumenConfigPath = filepath.Join(absRoot, LuumenConfigFile)
	state.RokitConfigPath = filepath.Join(absRoot, RokitConfigFile)
	state.WallyConfigPath = filepath.Join(absRoot, WallyConfigFile)

	state.HasLuumenConfig = fileExists(state.LuumenConfigPath)
	state.HasRokitConfig = fileExists(state.RokitConfigPath)
	state.HasWallyConfig = fileExists(state.WallyConfigPath)

	state.RojoProjectPaths, err = findRojoProjectFiles(absRoot)
	if err != nil {
		return Workspace{}, err
	}
	state.HasRojoProject = len(state.RojoProjectPaths) > 0

	state.IsLuumenManaged = state.HasLuumenConfig
	state.IsAdoptable = !state.IsLuumenManaged && (state.HasRokitConfig || state.HasWallyConfig || state.HasRojoProject)

	return state, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func findRojoProjectFiles(root string) ([]string, error) {
	globMatches, err := filepath.Glob(filepath.Join(root, "*.project.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to detect Rojo project files: %w", err)
	}

	paths := make([]string, 0, len(globMatches))
	seen := make(map[string]struct{}, len(globMatches))
	for _, match := range globMatches {
		if !fileExists(match) {
			continue
		}
		if _, alreadySeen := seen[match]; alreadySeen {
			continue
		}
		seen[match] = struct{}{}
		paths = append(paths, match)
	}

	sort.Strings(paths)
	return paths, nil
}
