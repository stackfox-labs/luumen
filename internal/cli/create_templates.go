package cli

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"luumen/internal/config"
)

//go:embed create_templates.json
var createTemplateCatalogFile []byte

type createTemplateCatalog struct {
	Templates []createTemplate `json:"templates"`
}

type createTemplate struct {
	Name             string                `json:"name"`
	Description      string                `json:"description"`
	Project          createTemplateProject `json:"project"`
	ScaffoldCommands []string              `json:"scaffoldCommands"`
	Directories      []string              `json:"directories"`
	Files            []createTemplateFile  `json:"files"`
}

type createTemplateProject struct {
	Install createTemplateInstall `json:"install"`
	Tasks   map[string][]string   `json:"tasks"`
}

type createTemplateInstall struct {
	Tools    bool `json:"tools"`
	Packages bool `json:"packages"`
}

type createTemplateFile struct {
	Path         string   `json:"path"`
	ContentLines []string `json:"contentLines"`
}

type createTemplateTokens struct {
	projectName string
	packageName string
}

type createTemplateCommandRunner func(command string, workingDir string) error

func loadBuiltinCreateTemplates() ([]createTemplate, error) {
	var catalog createTemplateCatalog
	if err := json.Unmarshal(createTemplateCatalogFile, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse create template catalog: %w", err)
	}

	if len(catalog.Templates) == 0 {
		return nil, fmt.Errorf("create template catalog is empty")
	}

	seen := make(map[string]struct{}, len(catalog.Templates))
	for _, template := range catalog.Templates {
		if err := validateCreateTemplate(template, seen); err != nil {
			return nil, err
		}
		seen[template.Name] = struct{}{}
	}

	return catalog.Templates, nil
}

func validateCreateTemplate(template createTemplate, seen map[string]struct{}) error {
	if strings.TrimSpace(template.Name) == "" {
		return fmt.Errorf("template name must not be empty")
	}
	if _, exists := seen[template.Name]; exists {
		return fmt.Errorf("duplicate template name %q", template.Name)
	}
	for _, command := range template.ScaffoldCommands {
		if strings.TrimSpace(command) == "" {
			return fmt.Errorf("template %q has empty scaffold command", template.Name)
		}
	}
	for _, dir := range template.Directories {
		if _, err := normalizeTemplateRelativePath(dir); err != nil {
			return fmt.Errorf("template %q has invalid directory %q: %w", template.Name, dir, err)
		}
	}
	for _, file := range template.Files {
		if _, err := normalizeTemplateRelativePath(file.Path); err != nil {
			return fmt.Errorf("template %q has invalid file path %q: %w", template.Name, file.Path, err)
		}
	}

	for name, steps := range template.Project.Tasks {
		if len(steps) == 0 {
			continue
		}
		if _, err := renderTaskSteps(steps, createTemplateTokens{}); err != nil {
			return fmt.Errorf("template %q has invalid task %q: %w", template.Name, name, err)
		}
	}

	return nil
}

func findCreateTemplate(templates []createTemplate, name string) (createTemplate, bool) {
	for _, template := range templates {
		if template.Name == name {
			return template, true
		}
	}
	return createTemplate{}, false
}

func createTemplateNames(templates []createTemplate) []string {
	names := make([]string, 0, len(templates))
	for _, template := range templates {
		names = append(names, template.Name)
	}
	sort.Strings(names)
	return names
}

func scaffoldProjectFromTemplate(targetPath string, projectName string, template createTemplate, runCommand createTemplateCommandRunner, writeConfig func(path string, cfg *config.Config) error) error {
	tokens := createTemplateTokens{
		projectName: projectName,
		packageName: normalizePackageName(projectName),
	}

	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	for _, command := range template.ScaffoldCommands {
		rendered := strings.TrimSpace(renderTemplateTokens(command, tokens))
		if rendered == "" {
			continue
		}
		if runCommand == nil {
			continue
		}
		if err := runCommand(rendered, targetPath); err != nil {
			return fmt.Errorf("failed scaffold command %q: %w", rendered, err)
		}
	}

	for _, relativeDir := range template.Directories {
		fullPath, err := resolveTemplatePath(targetPath, relativeDir)
		if err != nil {
			return fmt.Errorf("failed to resolve template directory %q: %w", relativeDir, err)
		}
		if err := os.MkdirAll(fullPath, 0o755); err != nil {
			return fmt.Errorf("failed to create template directory %q: %w", relativeDir, err)
		}
	}

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: projectName},
		Luu: config.LuuConfig{
			Install: config.InstallConfig{
				Tools:    template.Project.Install.Tools,
				Packages: template.Project.Install.Packages,
			},
		},
	}
	if len(template.Project.Tasks) > 0 {
		cfg.Tasks = make(map[string]config.TaskValue, len(template.Project.Tasks))
		for name, steps := range template.Project.Tasks {
			if len(steps) == 0 {
				continue
			}
			rendered, err := renderTaskSteps(steps, tokens)
			if err != nil {
				return fmt.Errorf("template task %q is invalid: %w", name, err)
			}
			if len(rendered) == 0 {
				continue
			}
			cfg.Tasks[name] = config.NewTaskValue(rendered...)
		}
		if len(cfg.Tasks) == 0 {
			cfg.Tasks = nil
		}
	}

	if err := writeConfig(filepath.Join(targetPath, config.FileName), cfg); err != nil {
		return fmt.Errorf("failed to write %s: %w", config.FileName, err)
	}

	for _, file := range template.Files {
		fullPath, err := resolveTemplatePath(targetPath, file.Path)
		if err != nil {
			return fmt.Errorf("failed to resolve template file %q: %w", file.Path, err)
		}
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return fmt.Errorf("failed to create parent directory for %q: %w", file.Path, err)
		}

		content := strings.Join(file.ContentLines, "\n")
		content = renderTemplateTokens(content, tokens)
		content = strings.TrimRight(content, "\n") + "\n"

		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write %q: %w", file.Path, err)
		}
	}

	return nil
}

func renderTaskSteps(steps []string, tokens createTemplateTokens) ([]string, error) {
	if len(steps) == 0 {
		return nil, fmt.Errorf("task step list must not be empty")
	}
	rendered := make([]string, 0, len(steps))
	for _, step := range steps {
		trimmed := strings.TrimSpace(renderTemplateTokens(step, tokens))
		if trimmed == "" {
			return nil, fmt.Errorf("task step must not be empty")
		}
		rendered = append(rendered, trimmed)
	}
	return rendered, nil
}

func renderTemplateTokens(input string, tokens createTemplateTokens) string {
	replacer := strings.NewReplacer(
		"{{project_name}}", tokens.projectName,
		"{{package_name}}", tokens.packageName,
	)
	return replacer.Replace(input)
}

func normalizeTemplateRelativePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("path is empty")
	}

	normalized := filepath.Clean(filepath.FromSlash(trimmed))
	if normalized == "." {
		return "", fmt.Errorf("path resolves to current directory")
	}
	if filepath.IsAbs(normalized) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	if normalized == ".." || strings.HasPrefix(normalized, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes target directory")
	}

	return normalized, nil
}

func resolveTemplatePath(root string, relative string) (string, error) {
	normalized, err := normalizeTemplateRelativePath(relative)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, normalized), nil
}
