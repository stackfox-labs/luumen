package cli

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"

	"luumen/internal/process"
	"luumen/internal/resolver"
	"luumen/internal/tools"
	"luumen/internal/workspace"
)

type addCommandDeps struct {
	detectWorkspace func(path string) (workspace.Workspace, error)
	resolve         func(input string) (resolver.Resolution, error)
	rokitInstaller  rokitInstaller
	wallyInstaller  wallyInstaller
}

func defaultAddCommandDeps() addCommandDeps {
	return addCommandDeps{
		detectWorkspace: workspace.Detect,
		resolve:         resolver.Resolve,
		rokitInstaller:  tools.NewRokit(nil, ""),
		wallyInstaller:  tools.NewWally(nil, ""),
	}
}

func newAddCmd(deps addCommandDeps) *cobra.Command {
	if deps.detectWorkspace == nil {
		deps.detectWorkspace = workspace.Detect
	}
	if deps.resolve == nil {
		deps.resolve = resolver.Resolve
	}
	if deps.rokitInstaller == nil {
		deps.rokitInstaller = tools.NewRokit(nil, "")
	}
	if deps.wallyInstaller == nil {
		deps.wallyInstaller = tools.NewWally(nil, "")
	}

	var noInstall bool

	cmd := &cobra.Command{
		Use:   "add <dependency>",
		Short: "Add a tool or package dependency",
		Long: "Add resolves natural inputs, explicit tool:/pkg: prefixes, and known tool aliases, " +
			"then mutates the corresponding ecosystem config file.",
		Example: "luu add rojo\n" +
			"luu add tool:rojo-rbx/rojo\n" +
			"luu add pkg:sleitnick/knit\n" +
			"luu add --no-install rojo",
		Args: requireExactlyOneArg("dependency"),
		RunE: func(cmd *cobra.Command, args []string) error {
			statusf(cmd, "Resolving dependency: %s", args[0])

			state, err := deps.detectWorkspace("")
			if err != nil {
				return fmt.Errorf("failed to detect workspace: %w. Next: run the command from a repository directory", err)
			}

			resolved, err := deps.resolve(args[0])
			if err != nil {
				if isResolverKind(err, resolver.ErrUnknownDependencyKind) {
					return fmt.Errorf("failed to resolve dependency kind: %w. Next: use tool:<owner/repo> or pkg:<scope/name>", err)
				}
				if isResolverKind(err, resolver.ErrAmbiguousDependency) {
					return fmt.Errorf("dependency resolution is ambiguous: %w. Next: use tool:<owner/repo> or pkg:<scope/name>", err)
				}
				return fmt.Errorf("failed to resolve dependency %q: %w", args[0], err)
			}

			switch resolved.Kind {
			case resolver.DependencyKindTool:
				if !state.HasRokitConfig {
					return fmt.Errorf("cannot add tool: %s is missing from %s. Next: add rokit.toml or run luu init in an adoptable repository", workspace.RokitConfigFile, state.RootPath)
				}
				statusf(cmd, "Updating %s with tool %s", workspace.RokitConfigFile, resolved.Value)
				if _, err := addToolToRokitConfig(state.RokitConfigPath, resolved.Value); err != nil {
					return fmt.Errorf("failed to update %s: %w", workspace.RokitConfigFile, err)
				}
				if noInstall {
					statusf(cmd, "Added tool without install (--no-install)")
					return nil
				}
				statusf(cmd, "Installing tools with Rokit...")
				if _, err := deps.rokitInstaller.Install(cmd.Context(), defaultToolRunOptions(cmd, state.RootPath)); err != nil {
					if process.IsKind(err, process.ErrorKindNotFound) {
						return fmt.Errorf("failed to install added tool: Rokit executable was not found in PATH: %w", err)
					}
					return fmt.Errorf("failed to install added tool via Rokit: %w", err)
				}
				statusf(cmd, "Tool added and installed successfully")
				return nil
			case resolver.DependencyKindPackage:
				if !state.HasWallyConfig {
					return fmt.Errorf("cannot add package: %s is missing from %s. Next: add wally.toml or run luu init in an adoptable repository", workspace.WallyConfigFile, state.RootPath)
				}
				statusf(cmd, "Updating %s with package %s", workspace.WallyConfigFile, resolved.Value)
				if _, err := addPackageToWallyConfig(state.WallyConfigPath, resolved.Value); err != nil {
					return fmt.Errorf("failed to update %s: %w", workspace.WallyConfigFile, err)
				}
				if noInstall {
					statusf(cmd, "Added package without install (--no-install)")
					return nil
				}
				statusf(cmd, "Installing packages with Wally...")
				if _, err := deps.wallyInstaller.Install(cmd.Context(), defaultToolRunOptions(cmd, state.RootPath)); err != nil {
					if process.IsKind(err, process.ErrorKindNotFound) {
						return fmt.Errorf("failed to install added package: Wally executable was not found in PATH: %w", err)
					}
					return fmt.Errorf("failed to install added package via Wally: %w", err)
				}
				statusf(cmd, "Package added and installed successfully")
				return nil
			default:
				return fmt.Errorf("failed to resolve dependency kind for %q", args[0])
			}
		},
	}

	cmd.Flags().BoolVar(&noInstall, "no-install", false, "skip install after mutating dependency config")
	return cmd
}

func isResolverKind(err error, kind error) bool {
	if err == nil || kind == nil {
		return false
	}
	return errors.Is(err, kind)
}

func addToolToRokitConfig(path string, toolRef string) (bool, error) {
	doc, err := readTomlDocument(path)
	if err != nil {
		return false, err
	}

	toolsTable, err := getOrCreateTable(doc, "tools")
	if err != nil {
		return false, err
	}

	if containsStringValue(toolsTable, toolRef) {
		return false, writeTomlDocument(path, doc)
	}

	key := sanitizeKey(lastPathSegment(toolRef))
	if key == "" {
		key = "tool"
	}
	key = nextAvailableKey(toolsTable, key)
	toolsTable[key] = toolRef

	doc["tools"] = toolsTable
	if err := writeTomlDocument(path, doc); err != nil {
		return false, err
	}

	return true, nil
}

func addPackageToWallyConfig(path string, pkgRef string) (bool, error) {
	doc, err := readTomlDocument(path)
	if err != nil {
		return false, err
	}

	depsTable, err := getOrCreateTable(doc, "dependencies")
	if err != nil {
		return false, err
	}

	if containsStringValue(depsTable, pkgRef) {
		return false, writeTomlDocument(path, doc)
	}

	pkgName := strings.Split(lastPathSegment(pkgRef), "@")[0]
	key := sanitizeKey(pkgName)
	if key == "" {
		key = "package"
	}
	key = nextAvailableKey(depsTable, key)
	depsTable[key] = pkgRef

	doc["dependencies"] = depsTable
	if err := writeTomlDocument(path, doc); err != nil {
		return false, err
	}

	return true, nil
}

func readTomlDocument(path string) (map[string]any, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	doc := make(map[string]any)
	if strings.TrimSpace(string(contents)) == "" {
		return doc, nil
	}

	if err := toml.Unmarshal(contents, &doc); err != nil {
		return nil, err
	}

	return doc, nil
}

func writeTomlDocument(path string, doc map[string]any) error {
	output, err := toml.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, output, 0o644)
}

func getOrCreateTable(doc map[string]any, key string) (map[string]any, error) {
	existing, ok := doc[key]
	if !ok {
		table := make(map[string]any)
		doc[key] = table
		return table, nil
	}

	table, ok := existing.(map[string]any)
	if ok {
		return table, nil
	}

	if typed, ok := existing.(map[string]string); ok {
		converted := make(map[string]any, len(typed))
		for k, v := range typed {
			converted[k] = v
		}
		doc[key] = converted
		return converted, nil
	}

	return nil, fmt.Errorf("invalid [%s] table type", key)
}

func containsStringValue(table map[string]any, value string) bool {
	for _, current := range table {
		currentString, ok := current.(string)
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(currentString), strings.TrimSpace(value)) {
			return true
		}
	}
	return false
}

func nextAvailableKey(table map[string]any, base string) string {
	if _, exists := table[base]; !exists {
		return base
	}
	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s%d", base, index)
		if _, exists := table[candidate]; !exists {
			return candidate
		}
	}
}

func lastPathSegment(value string) string {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) == 0 {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

var keySanitizer = regexp.MustCompile(`[^a-z0-9-_]+`)

func sanitizeKey(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	sanitized := keySanitizer.ReplaceAllString(lower, "-")
	sanitized = strings.Trim(sanitized, "-")
	return sanitized
}
