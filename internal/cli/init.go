package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"luumen/internal/config"
	"luumen/internal/process"
	"luumen/internal/tools"
	"luumen/internal/workspace"
)

type initCommandDeps struct {
	detectWorkspace func(path string) (workspace.Workspace, error)
	writeConfig     func(path string, cfg *config.Config) error
	rokitInstaller  rokitInstaller
	wallyInstaller  wallyInstaller
}

func defaultInitCommandDeps() initCommandDeps {
	return initCommandDeps{
		detectWorkspace: workspace.Detect,
		writeConfig:     config.Write,
		rokitInstaller:  tools.NewRokit(nil, ""),
		wallyInstaller:  tools.NewWally(nil, ""),
	}
}

func newInitCmd(deps initCommandDeps) *cobra.Command {
	if deps.detectWorkspace == nil {
		deps.detectWorkspace = workspace.Detect
	}
	if deps.writeConfig == nil {
		deps.writeConfig = config.Write
	}
	if deps.rokitInstaller == nil {
		deps.rokitInstaller = tools.NewRokit(nil, "")
	}
	if deps.wallyInstaller == nil {
		deps.wallyInstaller = tools.NewWally(nil, "")
	}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Adopt an existing repo into Luumen",
		Long: "Init inspects the current repository for Rokit, Wally, and Rojo files, " +
			"then generates project.config.luau with sensible default task mappings.",
		Example: "luu init\n" +
			"luu init --quiet",
		Args: requireNoPositionalArgs(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			statusf(cmd, "Inspecting repository for adoption...")
			spacef(cmd)
			reader := bufio.NewReader(cmd.InOrStdin())

			state, err := deps.detectWorkspace("")
			if err != nil {
				return fmt.Errorf("failed to detect workspace: %w. Next: run the command from a repository directory", err)
			}

			if state.HasLuumenConfig {
				return fmt.Errorf("%s already exists at %s; refusing to overwrite existing configuration", workspace.LuumenConfigFile, state.LuumenConfigPath)
			}

			if !state.HasRokitConfig && !state.HasWallyConfig && !state.HasRojoProject {
				confirmed, err := confirmCreateInPlace(cmd, reader, state.RootPath)
				if err != nil {
					return err
				}
				if !confirmed {
					return fmt.Errorf("init cancelled by user. Next: run luu create <name> or rerun luu init and confirm the prompt")
				}

				empty, err := directoryIsEmpty(state.RootPath)
				if err != nil {
					return fmt.Errorf("failed to inspect directory %s: %w", state.RootPath, err)
				}
				if !empty {
					createdBasic, err := maybeCreateBasicConfig(
						cmd,
						reader,
						state,
						deps.writeConfig,
						fmt.Sprintf("This directory already has files, so Luumen cannot scaffold a fresh project in %s without risking your existing work.", state.RootPath),
					)
					if err != nil {
						return err
					}
					if createdBasic {
						return nil
					}
					return fmt.Errorf("init cancelled by user. Next: run luu create <name> in a new directory or rerun luu init and choose the basic config fallback")
				}

				statusf(cmd, "No adoptable config found. Scaffolding current directory...")
				spacef(cmd)
				installConfig, err := scaffoldMinimalProject(state.RootPath, filepath.Base(state.RootPath), nil, deps.writeConfig)
				if err != nil {
					return fmt.Errorf("failed to scaffold project in %s: %w", state.RootPath, err)
				}

				if installConfig.Tools {
					statusf(cmd, "Installing tools with Rokit...")
					if _, err := deps.rokitInstaller.Install(cmd.Context(), defaultToolRunOptions(cmd, state.RootPath)); err != nil {
						if process.IsKind(err, process.ErrorKindNotFound) {
							return fmt.Errorf("project scaffolded but tool install failed: Rokit executable was not found in PATH: %w", err)
						}
						return fmt.Errorf("project scaffolded but tool install failed: %w", err)
					}
					successf(cmd, "Tools installed")
				}
				spacef(cmd)

				if installConfig.Packages {
					statusf(cmd, "Installing packages with Wally...")
					if _, err := deps.wallyInstaller.Install(cmd.Context(), defaultToolRunOptions(cmd, state.RootPath)); err != nil {
						if process.IsKind(err, process.ErrorKindNotFound) {
							return fmt.Errorf("project scaffolded but package install failed: Wally executable was not found in PATH: %w", err)
						}
						return fmt.Errorf("project scaffolded but package install failed: %w", err)
					}
					successf(cmd, "Packages installed")
				}

				if !installConfig.Tools && !installConfig.Packages {
					statusf(cmd, "Template does not define dependency installation")
				}

				statusf(cmd, "Initialized new Luumen project in %s", state.RootPath)
				nextStepsf(cmd, "Setup complete", "luu dev", "luu doctor")
				return nil
			}

			if !state.HasRojoProject || len(state.RojoProjectPaths) == 0 {
				createdBasic, err := maybeCreateBasicConfig(
					cmd,
					reader,
					state,
					deps.writeConfig,
					"Luumen could not generate default tasks confidently because no Rojo project file (*.project.json) was found.",
				)
				if err != nil {
					return err
				}
				if createdBasic {
					return nil
				}
				return fmt.Errorf("unable to generate default tasks confidently: no Rojo project file (*.project.json) found. Next: add a project file like default.project.json or rerun luu init and choose the basic config fallback")
			}

			rojoProjectPath, err := toRelativeConfigPath(state.RootPath, state.RojoProjectPaths[0])
			if err != nil {
				return fmt.Errorf("failed to resolve Rojo project path: %w", err)
			}

			cfg := &config.Config{
				Project: config.ProjectConfig{
					Name: filepath.Base(state.RootPath),
				},
				Install: config.InstallConfig{
					Tools:    state.HasRokitConfig,
					Packages: state.HasWallyConfig,
				},
				Tasks: map[string]config.TaskValue{
					"dev":    config.NewTaskValue(fmt.Sprintf("rojo sourcemap %s --output sourcemap.json", rojoProjectPath), fmt.Sprintf("rojo serve %s", rojoProjectPath)),
					"build":  config.NewTaskValue(fmt.Sprintf("rojo build %s --output build.rbxl", rojoProjectPath)),
					"lint":   config.NewTaskValue("selene src"),
					"format": config.NewTaskValue("stylua src"),
					"test":   config.NewTaskValue("lune run test"),
				},
			}

			if err := deps.writeConfig(state.LuumenConfigPath, cfg); err != nil {
				return fmt.Errorf("failed to write %s: %w", workspace.LuumenConfigFile, err)
			}

			statusf(cmd, "Generated %s", workspace.LuumenConfigFile)
			nextStepsf(cmd, "Adoption complete", "luu install", "luu dev")

			return nil
		},
	}

	return cmd
}

func confirmCreateInPlace(cmd *cobra.Command, reader *bufio.Reader, rootPath string) (bool, error) {
	writer := cmd.OutOrStdout()
	fmt.Fprintln(writer)
	fmt.Fprintf(writer, "%s No adoptable repository config was found in %s.\n", styleWarning(writer, "warning:"), rootPath)
	fmt.Fprintf(writer, "%s %s %s ", promptPrefix(writer), styleAccent(writer, "Create a new Luumen project in this directory?"), styleMuted(writer, "[y/N]:"))

	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("failed to read confirmation input: %w", err)
	}

	choice := strings.ToLower(strings.TrimSpace(line))
	fmt.Fprintln(writer)
	if errors.Is(err, io.EOF) && choice == "" {
		return false, fmt.Errorf("unable to adopt repository: no %s, %s, or Rojo project files were found. Next: run luu create <name> or rerun luu init interactively to confirm in-place scaffolding", workspace.RokitConfigFile, workspace.WallyConfigFile)
	}

	switch choice {
	case "y", "yes":
		return true, nil
	case "", "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid confirmation %q. Next: answer y or n", choice)
	}
}

func maybeCreateBasicConfig(cmd *cobra.Command, reader *bufio.Reader, state workspace.Workspace, writeConfig func(path string, cfg *config.Config) error, reason string) (bool, error) {
	empty, err := directoryIsEmpty(state.RootPath)
	if err != nil {
		return false, fmt.Errorf("failed to inspect directory %s: %w", state.RootPath, err)
	}
	if empty {
		return false, nil
	}

	confirmed, err := confirmCreateBasicConfig(cmd, reader, state.RootPath, reason)
	if err != nil {
		return false, err
	}
	if !confirmed {
		return false, nil
	}

	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name: filepath.Base(state.RootPath),
		},
		Install: config.InstallConfig{
			Tools:    state.HasRokitConfig,
			Packages: state.HasWallyConfig,
		},
	}
	if err := writeConfig(state.LuumenConfigPath, cfg); err != nil {
		return false, fmt.Errorf("failed to write basic %s: %w", workspace.LuumenConfigFile, err)
	}

	statusf(cmd, "Generated basic %s", workspace.LuumenConfigFile)
	if cfg.Install.Tools || cfg.Install.Packages {
		nextStepsf(cmd, "Basic setup complete", "luu install", "define tasks in project.config.luau", "luu doctor")
	} else {
		nextStepsf(cmd, "Basic setup complete", "define tasks in project.config.luau", "luu doctor")
	}
	return true, nil
}

func confirmCreateBasicConfig(cmd *cobra.Command, reader *bufio.Reader, rootPath string, reason string) (bool, error) {
	writer := cmd.OutOrStdout()
	fmt.Fprintln(writer)
	fmt.Fprintf(writer, "%s %s\n", styleWarning(writer, "warning:"), strings.TrimSpace(reason))
	fmt.Fprintf(writer, "%s %s %s ", promptPrefix(writer), styleAccent(writer, fmt.Sprintf("Create a basic %s in %s instead?", workspace.LuumenConfigFile, rootPath)), styleMuted(writer, "[y/N]:"))

	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("failed to read confirmation input: %w", err)
	}

	choice := strings.ToLower(strings.TrimSpace(line))
	fmt.Fprintln(writer)
	if errors.Is(err, io.EOF) && choice == "" {
		return false, fmt.Errorf("unable to continue init automatically. Next: rerun luu init interactively to create a basic %s or use luu create <name> in a new directory", workspace.LuumenConfigFile)
	}

	switch choice {
	case "y", "yes":
		return true, nil
	case "", "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid confirmation %q. Next: answer y or n", choice)
	}
}

func directoryIsEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func toRelativeConfigPath(rootPath string, targetPath string) (string, error) {
	relPath, err := filepath.Rel(rootPath, targetPath)
	if err != nil {
		return "", err
	}
	if relPath == "." {
		return filepath.ToSlash(filepath.Base(targetPath)), nil
	}
	return filepath.ToSlash(relPath), nil
}
