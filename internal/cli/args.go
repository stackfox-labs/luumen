package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func requireExactlyOneArg(name string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		switch len(args) {
		case 1:
			return nil
		case 0:
			return fmt.Errorf("missing required argument <%s>. Next: run %s --help", name, cmd.CommandPath())
		default:
			return fmt.Errorf("too many arguments: expected 1 <%s>, got %d. Next: run %s --help", name, len(args), cmd.CommandPath())
		}
	}
}

func requireNoPositionalArgs() cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}

		joined := strings.Join(args, " ")
		return fmt.Errorf("%s does not accept positional arguments: %q. Next: run %s --help", cmd.CommandPath(), joined, cmd.CommandPath())
	}
}
