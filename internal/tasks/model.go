package tasks

import (
	"errors"
	"fmt"
	"strings"

	"luumen/internal/config"
)

type ExecutionPlan struct {
	Steps []string
}

func NormalizeTaskValue(value config.TaskValue) (ExecutionPlan, error) {
	if len(value.Steps) == 0 {
		return ExecutionPlan{}, errors.New("task must contain at least one step")
	}

	steps := make([]string, 0, len(value.Steps))
	for index, step := range value.Steps {
		trimmed := strings.TrimSpace(step)
		if trimmed == "" {
			return ExecutionPlan{}, fmt.Errorf("step %d must not be empty", index)
		}
		steps = append(steps, trimmed)
	}

	return ExecutionPlan{Steps: steps}, nil
}
