package tools

import (
	"context"
	"strings"

	"luumen/internal/process"
)

const DefaultRojoExecutable = "rojo"

type Rojo struct {
	runner     CommandRunner
	executable string
}

func NewRojo(runner CommandRunner, executable string) *Rojo {
	if strings.TrimSpace(executable) == "" {
		executable = DefaultRojoExecutable
	}

	return &Rojo{
		runner:     withDefaultRunner(runner),
		executable: executable,
	}
}

func (r *Rojo) ProjectFiles(rootPath string) ([]string, error) {
	return findProjectFiles(rootPath)
}

func (r *Rojo) Serve(ctx context.Context, args []string, options RunOptions) (process.Result, error) {
	command := process.Command{Name: r.executable, Args: append([]string{"serve"}, args...)}
	return executeTool(ctx, r.runner, "rojo", command, options)
}

func (r *Rojo) Build(ctx context.Context, args []string, options RunOptions) (process.Result, error) {
	command := process.Command{Name: r.executable, Args: append([]string{"build"}, args...)}
	return executeTool(ctx, r.runner, "rojo", command, options)
}

func (r *Rojo) Sourcemap(ctx context.Context, args []string, options RunOptions) (process.Result, error) {
	command := process.Command{Name: r.executable, Args: append([]string{"sourcemap"}, args...)}
	return executeTool(ctx, r.runner, "rojo", command, options)
}
