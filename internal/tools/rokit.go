package tools

import (
	"context"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"luumen/internal/process"
)

const (
	DefaultRokitExecutable = "rokit"
	RokitConfigFile        = "rokit.toml"
)

type Rokit struct {
	runner     CommandRunner
	executable string
}

var readerIsTerminal = func(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}

	return term.IsTerminal(int(file.Fd()))
}

func NewRokit(runner CommandRunner, executable string) *Rokit {
	if strings.TrimSpace(executable) == "" {
		executable = DefaultRokitExecutable
	}

	return &Rokit{
		runner:     withDefaultRunner(runner),
		executable: executable,
	}
}

func (r *Rokit) HasConfig(rootPath string) (bool, error) {
	return hasConfigFile(rootPath, RokitConfigFile)
}

func (r *Rokit) Install(ctx context.Context, options RunOptions) (process.Result, error) {
	args := []string{"install"}
	if shouldSkipTrustCheck(options.Env) {
		args = append(args, "--no-trust-check")
	}

	command := process.Command{Name: r.executable, Args: args}
	result, err := executeTool(ctx, r.runner, "rokit", command, options)
	if err == nil {
		return result, nil
	}

	if !process.IsKind(err, process.ErrorKindExitCode) {
		return result, err
	}

	if options.Stdout != io.Discard || options.Stderr != io.Discard {
		return result, err
	}

	if !readerIsTerminal(options.Stdin) {
		return result, err
	}

	retryOptions := options
	retryOptions.Stdout = os.Stdout
	retryOptions.Stderr = os.Stderr
	return executeTool(ctx, r.runner, "rokit", command, retryOptions)
}

func shouldSkipTrustCheck(env map[string]string) bool {
	if value, ok := lookupEnvValue(env, "LUU_ROKIT_NO_TRUST_CHECK"); ok {
		return isTruthy(value)
	}

	if value, ok := lookupEnvValue(env, "CI"); ok {
		return isTruthy(value)
	}

	return false
}

func lookupEnvValue(env map[string]string, key string) (string, bool) {
	if env == nil {
		return "", false
	}

	value, ok := env[key]
	return value, ok
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (r *Rokit) Sync(ctx context.Context, options RunOptions) (process.Result, error) {
	return r.Install(ctx, options)
}
