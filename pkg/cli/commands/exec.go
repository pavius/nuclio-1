package commands

import (
	"os"
	"github.com/spf13/cobra"
	"fmt"
	"io"
	"bytes"
	"io/ioutil"
	"net/http"
	"strconv"
)

type ExecOptions struct {
	ClusterIP    string
	ContentType  string
	Url          string
	Method       string
	Body         string
	Headers      string
}

func NewCmdExec(copts *CommonOptions) *cobra.Command {
	var execOpts ExecOptions
	cmd := &cobra.Command{
		Use:     "exec function-name [-n namespace] [options]",
		Short:   "Execute/Invoke a Function",
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) < 1 {
				return fmt.Errorf("Missing function name")
			}

			name, err := FuncName2Resource(args[0])
			if err != nil {
				return err
			}

			err = invokeFunc(cmd.OutOrStdout(), copts, &execOpts, name)
			return err
		},
	}

	cmd.Flags().StringVarP(&execOpts.ClusterIP, "cluster-ip", "i", os.Getenv("REMOTE_CLUSTER_IP"), "Remote cluster IP")
	cmd.Flags().StringVarP(&execOpts.ContentType, "content-type", "c", "application/json", "HTTP Content Type")
	cmd.Flags().StringVarP(&execOpts.Url, "url", "u", "", "invocation URL")
	cmd.Flags().StringVarP(&execOpts.Method, "method", "m", "GET", "HTTP Method")
	cmd.Flags().StringVarP(&execOpts.Body, "body", "b", "", "Message body")
	cmd.Flags().StringVarP(&execOpts.Headers, "headers", "d", "", "HTTP headers (name=val1, ..)")
	return cmd
}

func invokeFunc(writer io.Writer, copts *CommonOptions, execOpts *ExecOptions, name string) error {

	_, functioncrClient, err := getKubeClient(copts)
	if err != nil {
		return err
	}

	fc, err := functioncrClient.Get(copts.Namespace, name)
	if err != nil {
		return err
	}

	port :=  strconv.Itoa(int(fc.Spec.HTTPPort))

	fullpath := "http://" + execOpts.ClusterIP + ":" + port + "/" + execOpts.Url
	fmt.Fprintf(writer, "Request Url: %s\n   Opts:%+v\n", fullpath, *execOpts)

	client := &http.Client{}
	var req *http.Request

	if execOpts.Method=="GET" {
		req, err = http.NewRequest(execOpts.Method, fullpath, nil)
	} else {
		req, err = http.NewRequest(execOpts.Method, fullpath, bytes.NewBuffer([]byte(execOpts.Body)))
	}
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", execOpts.ContentType)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	fmt.Fprintf(writer, "Stat: %s\nBody:\n", res.Status)

	htmlData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	fmt.Fprintf(writer, "%s", htmlData)
	return nil
}