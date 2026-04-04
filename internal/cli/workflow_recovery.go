package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"luumen/internal/process"
	"luumen/internal/resolver"
	"luumen/internal/tools"
	"luumen/internal/workspace"
)

type selfHealingShellRunner struct {
	cmd            commandContext
	commandName    string
	state          workspace.Workspace
	rokitInstaller rokitInstaller
	knownTools     map[string]string
	attempted      map[string]struct{}
}

type commandContext interface {
	OutOrStdout() io.Writer
	InOrStdin() io.Reader
	Context() context.Context
}

type rokitAdder interface {
	Add(ctx context.Context, tool string, alias string, options tools.RunOptions) (process.Result, error)
}

func newSelfHealingShellRunner(cmd commandContext, commandName string, state workspace.Workspace, rokitInstaller rokitInstaller) *selfHealingShellRunner {
	known := resolver.ToolAliases()
	if _, ok := known["lune"]; !ok {
		known["lune"] = "lune-org/lune@0.10.4"
	}
	if _, ok := known["lute"]; !ok {
		known["lute"] = "luau-lang/lute@0.1.0-nightly.20260327"
	}
	if _, ok := known["luau"]; !ok {
		known["luau"] = "luau-lang/luau@0.680.0"
	}

	return &selfHealingShellRunner{
		cmd:            cmd,
		commandName:    commandName,
		state:          state,
		rokitInstaller: rokitInstaller,
		knownTools:     known,
		attempted:      make(map[string]struct{}),
	}
}

func (r *selfHealingShellRunner) RunShell(ctx context.Context, command string, options process.Options) (process.Result, error) {
	executable, ok := firstExecutableToken(command)
	if ok {
		recovered, err := r.ensureExecutable(ctx, executable, options)
		if err != nil {
			return process.Result{ExitCode: -1}, err
		}
		if recovered {
			r.printRunning(command)
		}
	}

	return process.RunShell(ctx, command, options)
}

func (r *selfHealingShellRunner) ensureExecutable(ctx context.Context, executable string, options process.Options) (bool, error) {
	normalized := normalizeExecutableName(executable)
	if normalized == "" || isShellBuiltin(normalized) {
		return false, nil
	}
	recovered := false

	canonical, known := knownToolCanonical(normalized, r.knownTools)
	declared := false
	if known {
		declared = rokitConfigDeclaresTool(r.rokitConfigPath(), canonical)
	}

	if known && !declared {
		recoverAttempted, err := r.recoverKnownTool(ctx, normalized, canonical, false, options)
		if err != nil {
			return recovered, err
		}
		recovered = recovered || recoverAttempted
	}

	if executableExists(executable) {
		return recovered, nil
	}

	if !known {
		return recovered, fmt.Errorf("[luu] Command %q requires %q, but it is not installed.\n[luu] Luumen does not know how to install this tool automatically.", r.commandName, normalized)
	}

	recoverAttempted, err := r.recoverKnownTool(ctx, normalized, canonical, true, options)
	if err != nil {
		return recovered, err
	}
	recovered = recovered || recoverAttempted

	return recovered, nil
}

func (r *selfHealingShellRunner) recoverKnownTool(ctx context.Context, tool string, canonical string, treatAsMissing bool, options process.Options) (bool, error) {
	attemptKey := "declared:"
	if treatAsMissing {
		attemptKey = "missing:"
	}
	attemptKey += tool
	if _, attempted := r.attempted[attemptKey]; attempted {
		return false, nil
	}
	r.attempted[attemptKey] = struct{}{}

	rokitAvailable := executableExists(tools.DefaultRokitExecutable)
	declared := rokitConfigDeclaresTool(r.rokitConfigPath(), canonical)

	accepted, err := r.confirmRecovery(tool, rokitAvailable, declared)
	if err != nil {
		return false, err
	}
	if !accepted {
		if treatAsMissing {
			return false, fmt.Errorf("[luu] Command %q requires %q, but it is not installed. Next: install %q manually or rerun with --install-missing", r.commandName, tool, tool)
		}
		return false, fmt.Errorf("[luu] Command %q requires %q, but it is not declared in %s. Next: add it to [tools] or rerun with --install-missing", r.commandName, tool, workspace.RokitConfigFile)
	}

	if !rokitAvailable {
		r.printRunning(rokitBootstrapCommand())
		if err := bootstrapRokit(ctx, options); err != nil {
			return false, err
		}
	}

	if !declared {
		if adder, ok := r.rokitInstaller.(rokitAdder); ok {
			if err := r.ensureRokitConfigFile(); err != nil {
				return false, err
			}

			repoRef := canonicalToolBase(canonical)
			alias := normalizeRokitAlias(repoRef, canonicalToolExecutable(canonical))
			r.printRunning(rokitAddCommandForDisplay(repoRef, alias))

			addOptions := tools.RunOptions{
				WorkingDir: r.state.RootPath,
				Env:        collectToolEnv(),
				Stdout:     options.Stdout,
				Stderr:     options.Stderr,
				Stdin:      options.Stdin,
			}
			if _, err := adder.Add(ctx, repoRef, alias, addOptions); err != nil {
				return false, fmt.Errorf("failed to add %q via Rokit: %w", tool, err)
			}

			return true, nil
		}
	}

	if err := r.ensureRokitConfig(canonical, declared); err != nil {
		return false, err
	}

	installOptions := tools.RunOptions{
		WorkingDir: r.state.RootPath,
		Env:        collectToolEnv(),
		Stdout:     options.Stdout,
		Stderr:     options.Stderr,
		Stdin:      options.Stdin,
	}
	r.printRunning(rokitInstallCommandForDisplay(installOptions.Env))
	if _, err := r.rokitInstaller.Install(ctx, installOptions); err != nil {
		return false, fmt.Errorf("failed to install %q via Rokit: %w", tool, err)
	}

	return true, nil
}

func (r *selfHealingShellRunner) rokitConfigPath() string {
	if strings.TrimSpace(r.state.RokitConfigPath) != "" {
		return r.state.RokitConfigPath
	}
	return filepath.Join(r.state.RootPath, workspace.RokitConfigFile)
}

func (r *selfHealingShellRunner) confirmRecovery(tool string, rokitAvailable bool, declared bool) (bool, error) {
	if isInstallMissingCommandContext(r.cmd) || isYesCommandContext(r.cmd) {
		return true, nil
	}

	if !promptsAllowedCommandContext(r.cmd) {
		if !rokitAvailable {
			return false, fmt.Errorf("[luu] Command %q requires %q, but neither %q nor Rokit are installed.\n[luu] Non-interactive mode will not install dependencies automatically. Next: rerun with --install-missing or --yes", r.commandName, tool, tool)
		}
		return false, fmt.Errorf("[luu] Command %q requires %q, but it is not installed.\n[luu] Non-interactive mode will not install dependencies automatically. Next: rerun with --install-missing or --yes", r.commandName, tool)
	}

	writer := r.cmd.OutOrStdout()
	reader := bufio.NewReader(r.cmd.InOrStdin())
	if !rokitAvailable {
		r.printRecoveryWarningf("Command %q requires %q, but neither %q nor Rokit are installed.", r.commandName, tool, tool)
		response, err := readPromptLine(reader, writer, r.recoveryPromptf("Install Rokit and then install %q?", tool))
		if err != nil {
			return false, fmt.Errorf("failed to read dependency install confirmation: %w", err)
		}
		return promptAccepted(response, true), nil
	}

	if declared {
		r.printRecoveryWarningf("Command %q requires %q, but it is not installed.", r.commandName, tool)
		response, err := readPromptLine(reader, writer, r.recoveryPromptf("Install %q now with Rokit?", tool))
		if err != nil {
			return false, fmt.Errorf("failed to read dependency install confirmation: %w", err)
		}
		return promptAccepted(response, true), nil
	}

	r.printRecoveryWarningf("Command %q requires %q, but it is not declared in %s.", r.commandName, tool, workspace.RokitConfigFile)
	response, err := readPromptLine(reader, writer, r.recoveryPromptf("Add and install %q with Rokit now?", tool))
	if err != nil {
		return false, fmt.Errorf("failed to read dependency install confirmation: %w", err)
	}
	return promptAccepted(response, true), nil
}

func (r *selfHealingShellRunner) printRecoveryWarningf(format string, args ...any) {
	if !r.shouldRenderStatus() {
		return
	}

	writer := r.cmd.OutOrStdout()
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(writer, "%s %s\n", statusPrefix(writer), styleWarning(writer, message))
}

func (r *selfHealingShellRunner) recoveryPromptf(format string, args ...any) string {
	writer := r.cmd.OutOrStdout()
	question := fmt.Sprintf(format, args...)
	return fmt.Sprintf("%s %s %s ", statusPrefix(writer), styleWarning(writer, question), styleMuted(writer, "[Y/n]:"))
}

func (r *selfHealingShellRunner) ensureRokitConfig(canonical string, declared bool) error {
	path := r.rokitConfigPath()
	if !r.state.HasRokitConfig {
		if err := os.WriteFile(path, []byte("[tools]\n"), 0o644); err != nil {
			return fmt.Errorf("failed to create %s: %w", workspace.RokitConfigFile, err)
		}
		r.state.HasRokitConfig = true
		r.state.RokitConfigPath = path
	}
	if declared {
		return nil
	}

	if _, err := addToolToRokitConfig(path, canonical); err != nil {
		return fmt.Errorf("failed to update %s: %w", workspace.RokitConfigFile, err)
	}
	return nil
}

func (r *selfHealingShellRunner) ensureRokitConfigFile() error {
	if r.state.HasRokitConfig {
		return nil
	}

	path := r.rokitConfigPath()
	if err := os.WriteFile(path, []byte("[tools]\n"), 0o644); err != nil {
		return fmt.Errorf("failed to create %s: %w", workspace.RokitConfigFile, err)
	}
	r.state.HasRokitConfig = true
	r.state.RokitConfigPath = path
	return nil
}

func firstExecutableToken(command string) (string, bool) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return "", false
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return "", false
	}

	first := strings.TrimSpace(fields[0])
	first = strings.Trim(first, "\"'")
	if first == "" {
		return "", false
	}
	if strings.ContainsAny(first, "|&;<>") {
		return "", false
	}

	return first, true
}

func normalizeExecutableName(executable string) string {
	name := strings.TrimSpace(executable)
	name = strings.Trim(name, "\"'")
	name = strings.TrimPrefix(name, "./")
	name = strings.TrimPrefix(name, ".\\")
	name = strings.TrimSuffix(name, ".exe")
	return strings.ToLower(strings.TrimSpace(name))
}

func executableExists(name string) bool {
	_, err := exec.LookPath(strings.TrimSpace(name))
	return err == nil
}

func isShellBuiltin(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "echo", "cd", "set", "if", "for", "call", "exit", "rem", "dir", "type", "copy", "del", "cls", "mkdir", "rmdir", "pwd", "test", "export":
		return true
	default:
		return false
	}
}

func knownToolCanonical(executable string, aliases map[string]string) (string, bool) {
	name := strings.ToLower(strings.TrimSpace(executable))
	if canonical, ok := aliases[name]; ok {
		return canonical, true
	}

	for _, canonical := range aliases {
		base := canonicalToolExecutable(canonical)
		if base == name {
			return canonical, true
		}
	}

	return "", false
}

func canonicalToolExecutable(canonical string) string {
	trimmed := strings.TrimSpace(strings.ToLower(canonical))
	if index := strings.Index(trimmed, "@"); index >= 0 {
		trimmed = trimmed[:index]
	}
	segments := strings.Split(trimmed, "/")
	if len(segments) == 0 {
		return trimmed
	}
	return segments[len(segments)-1]
}

func canonicalToolBase(canonical string) string {
	trimmed := strings.TrimSpace(strings.ToLower(canonical))
	if index := strings.Index(trimmed, "@"); index >= 0 {
		trimmed = trimmed[:index]
	}
	return trimmed
}

func rokitConfigDeclaresTool(path string, canonical string) bool {
	doc, err := readTomlDocument(path)
	if err != nil {
		return false
	}
	rawTools, ok := doc["tools"]
	if !ok {
		return false
	}

	expectedKey := sanitizeKey(canonicalToolExecutable(canonical))
	if expectedKey == "" {
		return false
	}
	base := canonicalToolBase(canonical)
	switch typed := rawTools.(type) {
	case map[string]any:
		value, ok := typed[expectedKey]
		if !ok {
			return false
		}
		ref, ok := value.(string)
		if !ok {
			return false
		}
		return canonicalToolBase(ref) == base
	case map[string]string:
		ref, ok := typed[expectedKey]
		if !ok {
			return false
		}
		return canonicalToolBase(ref) == base
	}

	return false
}

func bootstrapRokit(ctx context.Context, options process.Options) error {
	command := rokitBootstrapCommand()
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("[luu] Rokit is required but was not found. Next: install Rokit manually and rerun this command")
	}

	if _, err := process.RunShell(ctx, command, options); err != nil {
		return fmt.Errorf("[luu] Rokit bootstrap failed: %w. Next: install Rokit manually and rerun this command", err)
	}

	return nil
}

func rokitBootstrapCommand() string {
	if runtime.GOOS == "windows" {
		return "powershell -NoProfile -ExecutionPolicy Bypass -Command \"iwr https://raw.githubusercontent.com/rojo-rbx/rokit/main/install.ps1 -UseBasicParsing | iex\""
	}
	return "curl -fsSL https://raw.githubusercontent.com/rojo-rbx/rokit/main/install.sh | sh"
}

func rokitInstallCommandForDisplay(env map[string]string) string {
	if shouldSkipTrustCheckForDisplay(env) {
		return "rokit install --no-trust-check"
	}
	return "rokit install"
}

func shouldSkipTrustCheckForDisplay(env map[string]string) bool {
	if value, ok := env["LUU_ROKIT_NO_TRUST_CHECK"]; ok && isTruthyFlag(value) {
		return true
	}
	if value, ok := env["CI"]; ok && isTruthyFlag(value) {
		return true
	}
	return false
}

func (r *selfHealingShellRunner) printRunning(command string) {
	if strings.TrimSpace(command) == "" {
		return
	}
	if !r.shouldRenderStatus() {
		return
	}

	writer := r.cmd.OutOrStdout()
	prefix := styleAccent(writer, "[luu]")
	fmt.Fprintf(writer, "%s %s %s\n\n", prefix, styleMuted(writer, "running:"), styleCommand(writer, shellStyleCommand(command)))
}

func (r *selfHealingShellRunner) shouldRenderStatus() bool {
	if r == nil || r.cmd == nil {
		return false
	}
	typed, ok := r.cmd.(*cobra.Command)
	if !ok || typed == nil {
		return true
	}
	return !isQuiet(typed)
}

func promptAccepted(response string, defaultYes bool) bool {
	choice := strings.ToLower(strings.TrimSpace(response))
	if choice == "" {
		return defaultYes
	}
	switch choice {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		return false
	}
}

func isYesCommandContext(cmd commandContext) bool {
	typed, ok := cmd.(*cobra.Command)
	if !ok || typed == nil {
		return false
	}
	return isYes(typed)
}

func isInstallMissingCommandContext(cmd commandContext) bool {
	typed, ok := cmd.(*cobra.Command)
	if !ok || typed == nil {
		return false
	}
	return isInstallMissing(typed)
}

func promptsAllowedCommandContext(cmd commandContext) bool {
	typed, ok := cmd.(*cobra.Command)
	if !ok || typed == nil {
		return false
	}
	return promptsAllowed(typed)
}
