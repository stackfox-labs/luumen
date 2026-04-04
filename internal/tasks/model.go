package tasks

import (
	"errors"
	"fmt"
	"strings"

	"luumen/internal/config"
)

type ExecutionPlan struct {
	Commands []string
}

func NormalizeTaskValue(value config.TaskValue) (ExecutionPlan, error) {
	if len(value.Commands) == 0 {
		return ExecutionPlan{}, errors.New("task must contain at least one command")
	}

	commands := make([]string, 0, len(value.Commands))
	for index, command := range value.Commands {
		trimmed := strings.TrimSpace(command)
		if trimmed == "" {
			return ExecutionPlan{}, fmt.Errorf("command %d must not be empty", index)
		}
		commands = append(commands, trimmed)
	}

	return ExecutionPlan{Commands: commands}, nil
}
