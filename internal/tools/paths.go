package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func hasConfigFile(rootPath string, fileName string) (bool, error) {
	if strings.TrimSpace(rootPath) == "" {
		return false, fmt.Errorf("workspace root path is required")
	}

	configPath := filepath.Join(rootPath, fileName)
	info, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to inspect %s: %w", configPath, err)
	}

	if info.IsDir() {
		return false, fmt.Errorf("expected a file but found a directory: %s", configPath)
	}

	return true, nil
}

func findProjectFiles(rootPath string) ([]string, error) {
	if strings.TrimSpace(rootPath) == "" {
		return nil, fmt.Errorf("workspace root path is required")
	}

	matches, err := filepath.Glob(filepath.Join(rootPath, "*.project.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate Rojo project file pattern: %w", err)
	}

	files := make([]string, 0, len(matches))
	for _, path := range matches {
		info, statErr := os.Stat(path)
		if statErr != nil || info.IsDir() {
			continue
		}
		files = append(files, path)
	}

	sort.Strings(files)
	return files, nil
}
