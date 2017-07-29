package cli


import (
	"github.com/nuclio/nuclio/pkg/zap"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/nuclio/nuclio/pkg/cli/commands"
	"os"
)

type CliOptions struct {
	Verbose         bool
	FunctionPath    string
	OutputType      string
	OutputName      string
	Version         string
	NuclioSourceDir string
	NuclioSourceURL string
	PushRegistry    string
}

func NewNuclioCLI() *cobra.Command {
	var options commands.CommonOptions
	var loggerLevel nucliozap.Level

	rootCmd := &cobra.Command{
		Use:   "nuclio-cli",
		Short: "nuclio CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error

			if options.Verbose {
				loggerLevel = nucliozap.DebugLevel
			} else {
				loggerLevel = nucliozap.InfoLevel
			}

			options.Logger, err = nucliozap.NewNuclioZap("cmd", loggerLevel)
			if err != nil {
				return errors.Wrap(err, "Failed to create logger")
			}

			return nil
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&options.Kubeconf, "kconf", "k", os.Getenv("KUBECONFIG"),
		"Path to Kubernetes config (admin.conf)")
	rootCmd.PersistentFlags().StringVarP(&options.Namespace, "namespace", "n", "default", "Kubernetes namespace")

	// link child commands
	rootCmd.AddCommand(
		commands.NewCmdGet(&options),
		commands.NewCmdBuild(&options),
		commands.NewCmdRun(&options),
		commands.NewCmdUpdate(&options),
		commands.NewCmdExec(&options),
		commands.NewCmdDel(&options),
	)

	// TODO: add CLI commends & implementation for Logs (function logs), Stats/Top, Version (print CLI version)

	return rootCmd
}

