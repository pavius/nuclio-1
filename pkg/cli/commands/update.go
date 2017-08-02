package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/nuclio/nuclio/pkg/functioncr"
)

func NewCmdUpdate(copts *CommonOptions) *cobra.Command {
	var funcOpts FuncOptions
	cmd := &cobra.Command{
		Use:     "update resource-type name[:ver] [-n namespace] [options]",
		Short:   "Update a resource (e.g. function)",
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) < 2 {
				return fmt.Errorf("Missing resource type and name, e.g. function myfunc")
			}

			if args[0]!="fu" && args[0]!="function" {
				return fmt.Errorf("unknown resource type %s - try 'function'",args[0])
			}

			name, err := FuncName2Resource(args[1])
			if err != nil {
				return err
			}

			err = updateFunc(copts, &funcOpts, name)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(),"The function %s was updated succesfuly\n",name)

			return nil
		},
	}

	cmd.Flags().BoolVar(&funcOpts.Publish, "publish", false, "Publish a function version")
	initBuildOptions(cmd, &funcOpts)
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

	err = updateBuildFromFlags(fc, funcOpts)
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

