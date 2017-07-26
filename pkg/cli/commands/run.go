package commands

import (
	"github.com/spf13/cobra"
	"fmt"
	"github.com/nuclio/nuclio/pkg/functioncr"
	"io/ioutil"
	"github.com/ghodss/yaml"
)

func NewCmdRun(copts *CommonOptions) *cobra.Command {
	var buildOpts BuildOptions
	var funcOpts FuncOptions
	var interactive bool
	cmd := &cobra.Command{
		Use:     "run function-name [-n namespace] [options]",
		Short:   "Deploy and Run a Function",
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

			// TODO: see if we need to build (e.g. -p not null or --image), if so run build & update func spec

			err = updateFuncFromFlags(&fc, &funcOpts)
			if err != nil {
				return err
			}

			err = createFunction(copts, &fc)

			return err
		},
	}

	initFileOption(cmd, copts)
	initBuildOptions(cmd, &buildOpts)
	initFuncOptions(cmd, &funcOpts)
	cmd.Flags().BoolVarP(&interactive,"interactive","i",false,"Submit Function Calls Interactively")
	return cmd
}


func initFuncFromFile(copts *CommonOptions) (functioncr.Function, error) {

	fc := functioncr.Function{}
	fc.TypeMeta.APIVersion = "nuclio.io/v1"
	fc.TypeMeta.Kind = "Function"
	fc.Namespace = "default"

	if copts.SpecFile == "" {
		return fc, nil
	}

	text, err := ioutil.ReadFile(copts.SpecFile)
	if err != nil {
		return fc, err
	}

	err = yaml.Unmarshal(text, &fc)
	if err != nil {
		return fc, err
	}

	return fc, nil
}

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