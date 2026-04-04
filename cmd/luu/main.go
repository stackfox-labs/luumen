package main

import (
	"fmt"
	"os"

	"luumen/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, cli.RenderCLIError(os.Stderr, err))
		os.Exit(1)
	}
}
