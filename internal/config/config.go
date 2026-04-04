package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

const FileName = "luumen.toml"

var ErrConfigNotFound = errors.New("luumen.toml not found")

type Config struct {
	Project  ProjectConfig
	Install  InstallConfig
	Commands map[string]TaskValue
	Tasks    map[string]TaskValue
}

type ProjectConfig struct {
	Name string `toml:"name,omitempty"`
}

type InstallConfig struct {
	Tools    bool `toml:"tools,omitempty"`
	Packages bool `toml:"packages,omitempty"`
}

type TaskValue struct {
	Commands []string
}

func NewTaskValue(commands ...string) TaskValue {
	copied := append([]string(nil), commands...)
	return TaskValue{Commands: copied}
}

func (v TaskValue) AsRawValue() any {
	switch len(v.Commands) {
	case 0:
		return ""
	case 1:
		return v.Commands[0]
	default:
		copied := append([]string(nil), v.Commands...)
		return copied
	}
}

type rawConfig struct {
	Project  ProjectConfig  `toml:"project,omitempty"`
	Install  InstallConfig  `toml:"install,omitempty"`
	Commands map[string]any `toml:"commands,omitempty"`
	Tasks    map[string]any `toml:"tasks,omitempty"`
}

func Load(path string) (*Config, error) {
	if path == "" {
		return nil, errors.New("config path is required")
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var raw rawConfig
	if err := toml.Unmarshal(contents, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filepath.Base(path), err)
	}

	cfg, err := fromRaw(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", filepath.Base(path), err)
	}

	return cfg, nil
}

func LoadFromDir(dir string) (*Config, error) {
	return Load(filepath.Join(dir, FileName))
}

func Write(path string, cfg *Config) error {
	if path == "" {
		return errors.New("config path is required")
	}
	if cfg == nil {
		return errors.New("config is nil")
	}

	raw, err := toRaw(cfg)
	if err != nil {
		return err
	}

	output, err := toml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("failed to encode %s: %w", filepath.Base(path), err)
	}

	if err := os.WriteFile(path, output, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return nil
}

func WriteToDir(dir string, cfg *Config) error {
	return Write(filepath.Join(dir, FileName), cfg)
}

func fromRaw(raw rawConfig) (*Config, error) {
	commands, err := normalizeTaskMap("commands", raw.Commands)
	if err != nil {
		return nil, err
	}

	tasks, err := normalizeTaskMap("tasks", raw.Tasks)
	if err != nil {
		return nil, err
	}

	return &Config{
		Project:  raw.Project,
		Install:  raw.Install,
		Commands: commands,
		Tasks:    tasks,
	}, nil
}

func toRaw(cfg *Config) (rawConfig, error) {
	commands, err := denormalizeTaskMap("commands", cfg.Commands)
	if err != nil {
		return rawConfig{}, err
	}

	tasks, err := denormalizeTaskMap("tasks", cfg.Tasks)
	if err != nil {
		return rawConfig{}, err
	}

	return rawConfig{
		Project:  cfg.Project,
		Install:  cfg.Install,
		Commands: commands,
		Tasks:    tasks,
	}, nil
}

func normalizeTaskMap(scope string, values map[string]any) (map[string]TaskValue, error) {
	if len(values) == 0 {
		return nil, nil
	}

	normalized := make(map[string]TaskValue, len(values))
	keys := sortedKeys(values)
	for _, key := range keys {
		task, err := parseTaskValue(values[key])
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", scope, key, err)
		}
		normalized[key] = task
	}

	return normalized, nil
}

func denormalizeTaskMap(scope string, values map[string]TaskValue) (map[string]any, error) {
	if len(values) == 0 {
		return nil, nil
	}

	raw := make(map[string]any, len(values))
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		task := values[key]
		normalized, err := normalizeCommandList(task.Commands)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", scope, key, err)
		}
		if len(normalized) == 1 {
			raw[key] = normalized[0]
			continue
		}
		raw[key] = normalized
	}

	return raw, nil
}

func parseTaskValue(value any) (TaskValue, error) {
	switch typed := value.(type) {
	case string:
		commands, err := normalizeCommandList([]string{typed})
		if err != nil {
			return TaskValue{}, err
		}
		return TaskValue{Commands: commands}, nil
	case []string:
		commands, err := normalizeCommandList(typed)
		if err != nil {
			return TaskValue{}, err
		}
		return TaskValue{Commands: commands}, nil
	case []any:
		commands := make([]string, 0, len(typed))
		for index, item := range typed {
			command, ok := item.(string)
			if !ok {
				return TaskValue{}, fmt.Errorf("array item %d must be a string", index)
			}
			commands = append(commands, command)
		}
		normalized, err := normalizeCommandList(commands)
		if err != nil {
			return TaskValue{}, err
		}
		return TaskValue{Commands: normalized}, nil
	default:
		return TaskValue{}, errors.New("expected a string or array of strings")
	}
}

func normalizeCommandList(commands []string) ([]string, error) {
	if len(commands) == 0 {
		return nil, errors.New("must contain at least one command")
	}

	normalized := make([]string, 0, len(commands))
	for _, command := range commands {
		trimmed := strings.TrimSpace(command)
		if trimmed == "" {
			return nil, errors.New("command must not be empty")
		}
		normalized = append(normalized, trimmed)
	}

	return normalized, nil
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
