package main

import (
	"fmt"
	"os"

	"github.com/lakekeeper/go-lakekeeper/cmd/lkctl/commands"
)

func main() {
	command := commands.NewCommand()

	command.SilenceErrors = true
	command.SilenceUsage = true

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
