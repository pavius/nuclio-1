package app

import (
	"github.com/nuclio/nuclio/pkg/cli"
)

func Run() error {
	cmd := cli.NewNuclioCLI()
	return cmd.Execute()
}
