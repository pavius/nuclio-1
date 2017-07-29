package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/nuclio/nuclio/pkg/cli"
	"flag"
)

func main() {
	path := flag.String("path", "./", "Path to write MD doc files")
	flag.Parse()

	cmd := cli.NewNuclioCLI()
	err := GenMarkdown(cmd, *path )
	if err != nil {
		panic(err)
	}


}

// Auto generate MD
func GenMarkdown(cmd *cobra.Command, path string) error {
	err := doc.GenMarkdownTree(cmd, path)
	if err != nil {
		return err
	}
	return nil
}

