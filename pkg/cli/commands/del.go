package commands

import (
	"github.com/spf13/cobra"
	"fmt"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdDel(copts *CommonOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "del resource-type name[:ver] [-n namespace] [options]",
		Short:   "Delete a Function",
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

			err = deleteFunction(copts, name)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(),"The function %s was deleted succesfuly\n",name)

			return nil
		},
	}

	return cmd
}

func deleteFunction(copts *CommonOptions, name string) error {
	_, functioncrClient, err := getKubeClient(copts)
	if err != nil {
		return err
	}

	err = functioncrClient.Delete(copts.Namespace, name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}