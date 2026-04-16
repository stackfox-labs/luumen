package main

import (
	"fmt"
	"os"

	"luumen/internal/cli"
)

func main() {
	cmd := cli.NewRootCmd()
	cmd.Use = "luu-dev"

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, cli.RenderCLIError(os.Stderr, err))
		os.Exit(1)
	}
}
