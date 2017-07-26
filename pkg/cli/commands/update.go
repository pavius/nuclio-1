package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/nuclio/nuclio/pkg/functioncr"
)

func NewCmdUpdate(copts *CommonOptions) *cobra.Command {
	var buildOpts BuildOptions
	var funcOpts FuncOptions
	cmd := &cobra.Command{
		Use:     "update function-name [-n namespace] [options]",
		Short:   "Update a Function",
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) < 1 {
				return fmt.Errorf("Missing function name")
			}

			name, err := FuncName2Resource(args[0])
			if err != nil {
				return err
			}

			err = updateFunc(copts, &funcOpts, name)
			return err
		},
	}

	cmd.Flags().BoolVar(&funcOpts.Publish, "publish", false, "Publish a function version")
	initBuildOptions(cmd, &buildOpts)
	initFuncOptions(cmd, &funcOpts)
	return cmd
}


func updateFunc(copts *CommonOptions, funcOpts *FuncOptions, name string) error {
	var fc *functioncr.Function

	_, functioncrClient, err := getKubeClient(copts)
	if err != nil {
		return err
	}

	fc, err = functioncrClient.Get(copts.Namespace, name)
	if err != nil {
		return err
	}

	// TODO: see if we need to build (e.g. -p not null or --image), if so run build & update func spec

	err = updateFuncFromFlags(fc, funcOpts)
	if err != nil {
		return err
	}

	if funcOpts.Publish && fc.Spec.Alias == "latest" {
		fc.Spec.Alias = ""
	}

	fc, err = functioncrClient.Update(fc)

	err = functioncr.WaitForFunctionProcessed(functioncrClient, fc.Namespace, fc.Name)
	if err != nil {
		return err
	}

	return nil
}

