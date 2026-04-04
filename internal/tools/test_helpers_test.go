package tools

import (
	"context"

	"luumen/internal/process"
)

type runCall struct {
	command process.Command
	options process.Options
}

type fakeRunner struct {
	calls  []runCall
	result process.Result
	err    error
}

func (f *fakeRunner) Run(_ context.Context, command process.Command, options process.Options) (process.Result, error) {
	f.calls = append(f.calls, runCall{command: command, options: options})
	return f.result, f.err
}
