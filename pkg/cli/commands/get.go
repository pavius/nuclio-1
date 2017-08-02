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
	"io"
)

type GetOptions struct {
	AllNamespaces  bool
	NotList        bool
	AnyVer         bool
	Watch          bool
	Labels         string
	Format         string
	ResType        string
	Resource       string
	Ver            string

}

func NewCmdGet(copts *CommonOptions) *cobra.Command {
	var getopts GetOptions
	cmd := &cobra.Command{
		Use:     "get resource-type [name[:version]] [-l selector] [-o text|wide|json|yaml] [--all-namespaces]",
		Short:   "Display one or many resources",
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) < 1 {
				return fmt.Errorf("Missing resource type, e.g. functions")
			}

			if args[0]!="fu" && args[0]!="functions" {
				return fmt.Errorf("unknown resource type %s - try 'function'",args[0])
			}

			// TODO: add more resource types (events ..) , for now support functions only

			if len(args) > 1 {
				err := parseName(args[1],&getopts)
				if err != nil {
					return err
				}
			}

			if getopts.AllNamespaces {
				copts.Namespace = ""
			}

			err := getFunction(cmd.OutOrStdout(), copts, &getopts)

			return err
		},
	}

	cmd.PersistentFlags().BoolVar(&getopts.AllNamespaces,"all-namespaces",false,"Show resources from all namespaces")
	cmd.Flags().StringVarP(&getopts.Labels, "labels", "l", "", "Label selector (lbl1=val1,lbl2=val2..)")
	cmd.Flags().StringVarP(&getopts.Format, "output", "o", "text", "Output format - text|wide|yaml|json")
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

func getFunction(writer io.Writer, copts *CommonOptions, getopts *GetOptions) error {

	// TODO: if list, filter functions by name and specified label selector

	clientSet, functioncrClient, err := getKubeClient(copts)
	if err != nil {
		return err
	}

	headers := []string{"Namespace", "Name", "Version", "State", "Local URL", "Host Port", "Replicas"}
	if getopts.Format=="wide" {
		headers = append(headers, "Labels")
	}

	if getopts.NotList {
		fc, err := functioncrClient.Get(copts.Namespace, getopts.Resource)
		if err != nil {
			return err
		}

		if getopts.Format=="text" || getopts.Format=="wide" {
			tp := render.NewTablePrinter(writer,headers)
			tp.Print(&[][]string{ getFunctionFields(clientSet, fc, getopts.Format=="wide") })
		} else {
			render.PrintInterface(writer, fc, getopts.Format)
		}

	} else {
		funcs, err := functioncrClient.List(copts.Namespace, meta_v1.ListOptions{LabelSelector:getopts.Labels})
		if err != nil {
			return err
		}

		if getopts.Format=="text" || getopts.Format=="wide" {
			tp := render.NewTablePrinter(writer, headers)
			data := [][]string{}
			for _,fc := range funcs.Items {
				data = append(data, getFunctionFields(clientSet, &fc, getopts.Format=="wide"))
			}
			tp.Print(&data)
		} else {
			render.PrintInterface(writer, funcs, getopts.Format)
		}

	}

	return nil

}

// return specific fields as string list for table printing
func getFunctionFields(cs *kubernetes.Clientset, fc *functioncr.Function, wide bool) []string {
	line := []string{fc.Namespace,fc.Labels["function"],fc.Labels["version"],string(fc.Status.State)}

	// add info from service & deployment
	// TODO: for lists we can get Service & Deployment info using .List get into a map to save http gets

	svc, err1 := cs.Core().Services(fc.Namespace).Get(fc.Name, meta_v1.GetOptions{})
	dep, err2 := cs.AppsV1beta1().Deployments(fc.Namespace).Get(fc.Name, meta_v1.GetOptions{})
	if err1 == nil && err2==nil {
		cport := strconv.Itoa(int(svc.Spec.Ports[0].Port))
		nport := strconv.Itoa(int(svc.Spec.Ports[0].NodePort))
		pods  := strconv.Itoa(int(dep.Status.AvailableReplicas))+"/"+strconv.Itoa(int(*dep.Spec.Replicas))
		line = append( line, []string{svc.Spec.ClusterIP+":"+cport, nport, pods}...)
	} else {
		line = append(line, []string{"-", "-", "-"}...)
	}
	if wide {
		line = append(line, render.Map2Str(fc.Labels))
	}
	return line
}