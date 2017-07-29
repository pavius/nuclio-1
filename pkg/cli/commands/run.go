package commands

import (
	"github.com/spf13/cobra"
	"fmt"
	"github.com/nuclio/nuclio/pkg/functioncr"
)

func NewCmdRun(copts *CommonOptions) *cobra.Command {
	var funcOpts FuncOptions
	//var interactive bool
	cmd := &cobra.Command{
		Use:     "run function-name [-n namespace] [options]",
		Short:   "Build, Deploy and Run a Function",
		Example: "run myfunc -p ./ -h HandleEvent",
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

			err = updateFuncFromFlags(&fc, &funcOpts)
			if err != nil {
				return err
			}

			err = createFunction(copts, &fc)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(),"The function %s was created succesfuly\n",fc.Name)

			return nil
		},
	}

	initFileOption(cmd, copts)
	initBuildOptions(cmd, &funcOpts)
	initFuncOptions(cmd, &funcOpts)
	//cmd.Flags().BoolVarP(&interactive,"interactive","t",false,"Submit Function Calls Interactively")
	return cmd
}

// create the function resource in Kubernetes and wait for controller confirmation
func createFunction(copts *CommonOptions, fc *functioncr.Function) error {
	_, functioncrClient, err := getKubeClient(copts)
	if err != nil {
		return err
	}

	fc, err = functioncrClient.Create(fc)
	if err != nil {
		return err
	}

	err = functioncr.WaitForFunctionProcessed(functioncrClient, fc.Namespace, fc.Name)
	if err != nil {
		return err
	}

	return nil
}