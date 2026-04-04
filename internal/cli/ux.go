package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"luumen/internal/process"
	"luumen/internal/tools"
)

type quietContextKey struct{}
type verboseContextKey struct{}

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
	return styleAccent(writer, "•")
}

func successPrefix(writer io.Writer) string {
	return styleSuccess(writer, "✓")
}

func promptPrefix(writer io.Writer) string {
	return styleAccent(writer, "◇")
}

func nextPrefix(writer io.Writer) string {
	return styleAccent(writer, "→")
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
