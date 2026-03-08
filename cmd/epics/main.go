package main

import (
	"os"

	"github.com/agentepics/epics.sh/internal/cli"
)

func main() {
	os.Exit(cli.NewApp("", os.Stdin, os.Stdout, os.Stderr).Run(os.Args[1:]))
}
