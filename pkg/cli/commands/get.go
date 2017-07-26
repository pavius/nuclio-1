package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"strings"
	"strconv"
	"github.com/nuclio/nuclio/pkg/functioncr"
	"k8s.io/client-go/kubernetes"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/nuclio/nuclio/pkg/cli/render"
)

type GetOptions struct {
	NotList   bool
	AnyVer    bool
	Watch     bool
	Labels    string
	Format    string
	Rtype     string
	Resource  string
	Ver       string

}

func NewCmdGet(copts *CommonOptions) *cobra.Command {
	var getopts GetOptions
	cmd := &cobra.Command{
		Use:     "get [resource-name[:version]] [-l selector] [-o text|json|yaml]",
		Short:   "Get/List resource attributes",
		RunE: func(cmd *cobra.Command, args []string) error {

			// TODO: add more resource types (events ..) , for now support functions only

			if len(args) > 0 {
				err := parseName(args[0],&getopts)
				if err != nil {
					return err
				}
			}

			err := getFunction(copts, &getopts, cmd)

			return err
		},
	}

	cmd.Flags().StringVarP(&getopts.Labels, "labels", "l", "", "Label selector (lbl1=val1,lbl2=val2..)")
	cmd.Flags().StringVarP(&getopts.Format, "output", "o", "text", "Output format - text|yaml|json")
	cmd.Flags().BoolVarP(&getopts.Watch,"watch","w",false,"Watch for changes")
	return cmd
}


func parseName(fullname string, opts *GetOptions) error {
	list := strings.Split(fullname,":")
	if len(list) == 1 {
		opts.Resource = list[0]
		opts.AnyVer = true
		return nil
	}
	_, err := strconv.Atoi(list[1])
	if list[1] == "latest" || err == nil {
		opts.NotList = true
		opts.Ver = list[1]
		if list[1] == "latest" {
			opts.Resource = list[0]
		} else {
			opts.Resource = list[0]+"-"+list[1]
		}
		return nil
	}
	return fmt.Errorf("Version name must be 'latest' or number - %s",err)
}

func getFunction(copts *CommonOptions, getopts *GetOptions, cmd *cobra.Command) error {


	clientSet, functioncrClient, err := getKubeClient(copts)
	if err != nil {
		return err
	}

	if getopts.NotList {
		fc, err := functioncrClient.Get(copts.Namespace, getopts.Resource)
		if err != nil {
			return err
		}

		if getopts.Format=="text" {
			tp := render.NewTablePrinter(cmd.OutOrStdout(),[]string{"Namespace", "Name", "Version", "State", "Local URL", "Host Port", "Replicas"})
			tp.Print(&[][]string{ getFunctionFields(clientSet, fc) })
		} else {
			render.PrintInterface(cmd.OutOrStdout(), fc, getopts.Format)
		}

	} else {
		funcs, err := functioncrClient.List(copts.Namespace, meta_v1.ListOptions{})
		if err != nil {
			return err
		}

		if getopts.Format=="text" {
			tp := render.NewTablePrinter(cmd.OutOrStdout(),[]string{"Namespace", "Name", "Version", "State", "Local URL", "Host Port", "Replicas"})
			data := [][]string{}
			for _,fc := range funcs.Items {
				data = append(data, getFunctionFields(clientSet, &fc))
			}
			tp.Print(&data)
		} else {
			render.PrintInterface(cmd.OutOrStdout(), funcs, getopts.Format)
		}

	}

	return nil

}

func getFunctionFields(cs *kubernetes.Clientset, fc *functioncr.Function) []string {
	line := []string{fc.Namespace,fc.Labels["function"],fc.Labels["version"],string(fc.Status.State)}

	// add info from service & deployment
	svc, err1 := cs.Core().Services(fc.Namespace).Get(fc.Name, meta_v1.GetOptions{})
	dep, err2 := cs.AppsV1beta1().Deployments(fc.Namespace).Get(fc.Name, meta_v1.GetOptions{})
	if err1 == nil && err2==nil {
		cport := strconv.Itoa(int(svc.Spec.Ports[0].Port))
		nport := strconv.Itoa(int(svc.Spec.Ports[0].NodePort))
		pods := strconv.Itoa(int(dep.Status.AvailableReplicas))+"/"+strconv.Itoa(int(*dep.Spec.Replicas))
		return append(line, []string{svc.Spec.ClusterIP+":"+cport, nport, pods}...)
	}
	return append(line, []string{"-", "-", "-"}...)
}