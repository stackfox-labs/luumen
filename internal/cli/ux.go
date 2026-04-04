package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"luumen/internal/process"
	"luumen/internal/tools"
)

type quietContextKey struct{}
type verboseContextKey struct{}
type yesContextKey struct{}
type noPromptContextKey struct{}
type installMissingContextKey struct{}

func withQuietMode(ctx context.Context, quiet bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, quietContextKey{}, quiet)
}

func withVerboseMode(ctx context.Context, verbose bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, verboseContextKey{}, verbose)
}

func withYesMode(ctx context.Context, yes bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, yesContextKey{}, yes)
}

func withNoPromptMode(ctx context.Context, noPrompt bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, noPromptContextKey{}, noPrompt)
}

func withInstallMissingMode(ctx context.Context, installMissing bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, installMissingContextKey{}, installMissing)
}

func isQuiet(cmd *cobra.Command) bool {
	if cmd == nil || cmd.Context() == nil {
		return false
	}
	value := cmd.Context().Value(quietContextKey{})
	quiet, ok := value.(bool)
	return ok && quiet
}

func isVerbose(cmd *cobra.Command) bool {
	if cmd == nil || cmd.Context() == nil {
		return false
	}
	value := cmd.Context().Value(verboseContextKey{})
	verbose, ok := value.(bool)
	return ok && verbose
}

func isYes(cmd *cobra.Command) bool {
	if cmd == nil || cmd.Context() == nil {
		return false
	}
	value := cmd.Context().Value(yesContextKey{})
	yes, ok := value.(bool)
	return ok && yes
}

func isNoPrompt(cmd *cobra.Command) bool {
	if cmd == nil || cmd.Context() == nil {
		return false
	}
	value := cmd.Context().Value(noPromptContextKey{})
	noPrompt, ok := value.(bool)
	return ok && noPrompt
}

func isInstallMissing(cmd *cobra.Command) bool {
	if cmd == nil || cmd.Context() == nil {
		return false
	}
	value := cmd.Context().Value(installMissingContextKey{})
	installMissing, ok := value.(bool)
	return ok && installMissing
}

func promptsAllowed(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	if isNoPrompt(cmd) {
		return false
	}
	if isTruthyFlag(os.Getenv("CI")) {
		return false
	}
	return isTerminalReader(cmd.InOrStdin())
}

func isTruthyFlag(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func isTerminalReader(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func shellStyleCommand(command string) string {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return ""
	}
	return trimmed
}

func statusf(cmd *cobra.Command, format string, args ...any) {
	if isQuiet(cmd) {
		return
	}
	message := fmt.Sprintf(format, args...)
	writer := cmd.OutOrStdout()
	fmt.Fprintf(writer, "%s %s\n", statusPrefix(writer), message)
}

func successf(cmd *cobra.Command, format string, args ...any) {
	if isQuiet(cmd) {
		return
	}
	message := fmt.Sprintf(format, args...)
	writer := cmd.OutOrStdout()
	fmt.Fprintf(writer, "%s %s\n", successPrefix(writer), message)
}

func spacef(cmd *cobra.Command) {
	if isQuiet(cmd) {
		return
	}
	fmt.Fprintln(cmd.OutOrStdout())
}

func nextStepsf(cmd *cobra.Command, title string, steps ...string) {
	if isQuiet(cmd) {
		return
	}

	writer := cmd.OutOrStdout()
	fmt.Fprintln(writer)
	if title != "" {
		fmt.Fprintf(writer, "%s %s\n", successPrefix(writer), title)
	}
	for _, step := range steps {
		if step == "" {
			continue
		}
		fmt.Fprintf(writer, "%s %s\n", nextPrefix(writer), step)
	}
}

func statusPrefix(writer io.Writer) string {
	return styleAccent(writer, "[luu]")
}

func successPrefix(writer io.Writer) string {
	return styleSuccess(writer, "[ok]")
}

func promptPrefix(writer io.Writer) string {
	return styleAccent(writer, "◇")
}

func nextPrefix(writer io.Writer) string {
	return styleAccent(writer, "[next]")
}

func commandOutputWriters(cmd *cobra.Command) (io.Writer, io.Writer) {
	if isVerbose(cmd) {
		return cmd.OutOrStdout(), cmd.ErrOrStderr()
	}

	return io.Discard, io.Discard
}

func defaultToolLogger(cmd *cobra.Command) tools.InvocationLogger {
	return func(toolName string, command process.Command) {
		if !isVerbose(cmd) {
			return
		}
		statusf(cmd, "Running %s command: %s", styleAccent(cmd.OutOrStdout(), toolName), styleCommand(cmd.OutOrStdout(), command.String()))
	}
}

func defaultToolRunOptions(cmd *cobra.Command, workingDir string) tools.RunOptions {
	stdout, stderr := commandOutputWriters(cmd)
	return tools.RunOptions{
		WorkingDir: workingDir,
		Env:        collectToolEnv(),
		Logger:     defaultToolLogger(cmd),
		Stdout:     stdout,
		Stderr:     stderr,
		Stdin:      cmd.InOrStdin(),
	}
}

func collectToolEnv() map[string]string {
	env := make(map[string]string)
	for _, key := range []string{"CI", "LUU_ROKIT_NO_TRUST_CHECK"} {
		if value, ok := os.LookupEnv(key); ok {
			env[key] = value
		}
	}

	if len(env) == 0 {
		return nil
	}

	return env
}
