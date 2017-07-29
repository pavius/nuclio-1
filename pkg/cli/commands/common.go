package commands

import (
	"github.com/spf13/cobra"
	"strings"
	"github.com/nuclio/nuclio/pkg/functioncr"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes"
	"k8s.io/api/core/v1"
	"github.com/pkg/errors"
	"strconv"
	"fmt"
	"github.com/nuclio/nuclio/pkg/zap"
	"os"
	"io/ioutil"
	"github.com/ghodss/yaml"
)


type CommonOptions struct {
	Verbose         bool
	Namespace       string
	Kubeconf        string
	KubeHost        string
	SpecFile        string
	Logger          *nucliozap.NuclioZap
}

type FuncOptions struct {
	CodePath        string
	CodeKey         string
	CodeWatch       bool
	Runtime         string
	Handler         string
	OutputType      string
	OutputName      string
	Version         string
	NuclioSourceDir string
	NuclioSourceURL string
	PushRegistry    string

	Description  string
	Image        string
	Env          string
	Labels       string
	Cpu          string
	Memory       string
	WorkDir      string
	Role         string
	Secret       string
	Events       string
	Data         string
	Disabled     bool
	Publish      bool
	HttpPort     int32
	Scale        string
	MinReplicas  int32
	MaxReplicas  int32
}

func initFileOption(cmd *cobra.Command, opts *CommonOptions) {
	cmd.Flags().StringVarP(&opts.SpecFile, "file", "f", "", "Function Spec File")
}

func initBuildOptions(cmd *cobra.Command, opts *FuncOptions) {
	cmd.Flags().StringVarP(&opts.CodePath, "path", "p", "", "Function source code path")
	cmd.Flags().StringVar(&opts.Handler, "handler", "handler", "Function handler name")
	cmd.Flags().StringVarP(&opts.Image, "image", "i", "", "Container image to use, will use function name if not specified")
	cmd.Flags().StringVar(&opts.Runtime, "runtime", "go", "Function runtime language and version e.g. go, python 2.7, ..")

	cmd.Flags().StringVarP(&opts.OutputType, "output", "o", "docker", "Build output type - docker|binary")
	cmd.Flags().StringVarP(&opts.PushRegistry, "registry", "r", os.Getenv("PUSH_REGISTRY"), "URL of container registry (for push)")
	cmd.Flags().StringVar(&opts.NuclioSourceDir, "src-dir", "", "Local directory with nuclio sources (avoid cloning) ")
	cmd.Flags().StringVar(&opts.NuclioSourceURL, "src-url", "git@github.com:nuclio/nuclio.git", "nuclio sources url for git clone")
}

func initFuncOptions(cmd *cobra.Command, opts *FuncOptions) {
	cmd.Flags().StringVar(&opts.Description, "desc", "", "Function description")
	cmd.Flags().StringVarP(&opts.Scale, "scale", "s", "1", "Function scaling (auto|number)")
	cmd.Flags().StringVarP(&opts.Labels, "labels", "l", "", "Additional function labels (lbl1=val1,lbl2=val2..)")
	cmd.Flags().StringVarP(&opts.Env, "env", "e", "", "Environment variables (name1=val1,name2=val2..)")
	cmd.Flags().StringVar(&opts.Events, "events", "", "Comma seperated list of event sources (in json)")
	cmd.Flags().StringVar(&opts.Data, "data", "", "Comma seperated list of data bindings (in json)")
	cmd.Flags().BoolVarP(&opts.Disabled, "disabled", "d", false, "Start function disabled (don't run yet)")
	cmd.Flags().Int32Var(&opts.HttpPort, "port", 0, "Public HTTP port (node port)")
	cmd.Flags().Int32Var(&opts.MinReplicas, "min-replica", 0, "Minimum number of function replicas")
	cmd.Flags().Int32Var(&opts.MaxReplicas, "max-replica", 0, "Maximum number of function replicas")
}

func getKubeClient(copts *CommonOptions) (*kubernetes.Clientset, *functioncr.Client, error) {
	restConfig, err := clientcmd.BuildConfigFromFlags("", copts.Kubeconf)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create REST config")
	}
	copts.KubeHost = restConfig.Host

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create client set")
	}

	// create a client for function custom resources
	functioncrClient, err := functioncr.NewClient(copts.Logger,
		restConfig,
		clientSet)

	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create function custom resource client")
	}

	return clientSet, functioncrClient, nil
}

// Read function spec file (if specified -f flag) and initialize defaults
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



func updateBuildFromFlags(fc *functioncr.Function, opts *FuncOptions) error {

	if opts.CodePath !="" {
		fc.Spec.Code.Path = opts.CodePath
	}

	if opts.Handler !="" {
		fc.Spec.Handler = opts.Handler
	}

	if opts.Runtime !="" {
		fc.Spec.Runtime = opts.Runtime
	}

	if opts.Image !="" {
		fc.Spec.Image = opts.Image
	}

	return nil
}

// merge existing/from-file function spec with command line options
func updateFuncFromFlags(fc *functioncr.Function, opts *FuncOptions) error {

	if opts.Description !="" {
		fc.Spec.Description = opts.Description
	}

	// update replicas if scale was specified
	if opts.Scale !="" {

		// TODO: handle/Set Min/Max replicas (used only with Auto mode)
		if opts.Scale == "auto" {
			fc.Spec.Replicas = 0
		} else {
			i, err := strconv.Atoi(opts.Scale)
			if err != nil {
				return fmt.Errorf("illegal function scale, must be 'auto' or an integer value ")
			} else {
				fc.Spec.Replicas = int32(i)
			}
		}
	}

	// Set specified labels, is label = "" remove it (if exists)
	labels := Str2Map(opts.Labels, ",")
	for k, v := range labels {
		if k != "function" && k!= "version" && k!="alias" {
			if v == "" {
				delete(fc.Labels, k)
			} else {
				fc.Labels[k] = v
			}
		}
	}

	envmap := Str2Map(opts.Env, ",")
	newenv := []v1.EnvVar{}

	// merge new Environment var: update existing then add new
	for _, e := range fc.Spec.Env {
		if v, ok := envmap[e.Name]; ok {
			if v != "" {
				newenv = append(newenv, v1.EnvVar{ Name:e.Name, Value:v})
			}
			delete(envmap, e.Name)
		} else {
			newenv = append(newenv, e)
		}
	}

	for k, v := range envmap {
		newenv = append(newenv, v1.EnvVar{ Name:k, Value:v})
	}
	fc.Spec.Env = newenv

	// TODO: update events and data

	if opts.HttpPort !=0 {
		fc.Spec.HTTPPort = opts.HttpPort
	}

	if opts.Publish {
		fc.Spec.Publish = opts.Publish
	}

	if opts.Disabled {
		fc.Spec.Disabled = opts.Disabled  // TODO: use string to detect if noop/true/false
	}
	return nil
}

func Str2Map(str,sep string) map[string]string {
	strsp := strings.Split(str,sep)
	strmap := make(map[string]string)
	for _,v := range strsp {
		kv := strings.Split(v,"=")
		if len(kv) > 1 {
			strmap[kv[0]] = kv[1]
		}
	}
	return strmap
}

func FuncName2Resource(fullname string) (string, error) {
	list := strings.Split(fullname,":")
	if len(list) == 1 {
		return fullname, nil
	}
	_, err := strconv.Atoi(list[1])
	if list[1] == "latest" || err == nil {
		if list[1] == "latest" {
			return list[0], nil
		} else {
			return list[0]+"-"+list[1], nil
		}
	}
	return "", fmt.Errorf("Version name must be 'latest' or number - %s",err)
}
