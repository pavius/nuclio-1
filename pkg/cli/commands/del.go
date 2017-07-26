package commands

import (
	"github.com/spf13/cobra"
	"fmt"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCmdDel(copts *CommonOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "del function-name [-n namespace] [options]",
		Short:   "Delete a Function",
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) < 1 {
				return fmt.Errorf("Missing function name")
			}

			name, err := FuncName2Resource(args[0])
			if err != nil {
				return err
			}

			err = deleteFunction(copts, name)
			return err
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