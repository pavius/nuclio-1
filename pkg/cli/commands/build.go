package commands

import (
	"github.com/spf13/cobra"
	"fmt"
)

func NewCmdBuild(copts *CommonOptions) *cobra.Command {
	var funcOpts FuncOptions
	cmd := &cobra.Command{
		Use:     "build function-name [-n namespace] [options]",
		Short:   "Build only for a Function (compile, prepare and push an image)",
		RunE: func(cmd *cobra.Command, args []string) error {

			fc, err := initFuncFromFile(copts)
			if err != nil {
				return err
			}

			if fc.Name == "" && len(args) < 1 {
				return fmt.Errorf("Missing function name")
			}

			if len(args) >= 1 {
				fc.Name = args[0]
			}

			if copts.Namespace != "" {
				fc.Namespace = copts.Namespace
			}

			err = updateBuildFromFlags(&fc, &funcOpts)
			if err != nil {
				return err
			}

			// TODO: see if we need to build (e.g. -p not null or --image), if so run build & update func spec
			// update code fields (code, handler, runtime, ..)


			return nil
		},
	}

	initFileOption(cmd, copts)
	initBuildOptions(cmd, &funcOpts)
	return cmd
}


