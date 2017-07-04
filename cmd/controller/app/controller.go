package app

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/nuclio/nuclio/pkg/logger"
	"github.com/nuclio/nuclio/pkg/kubecr"
)

type Controller struct {
	logger        logger.Logger
	confPath      string
}

func NewProcessor(configurationPath string) (*Controller, error) {

	newController := Controller{ confPath:configurationPath }

	return &newController, nil
}

func (p *Controller) Start() error {

	config, err := getClientConfig(p.confPath)
	if err != nil {
		return err
	}
	cl, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	tprcl, err := createFuncCR(config, cl)
	if err != nil {
		return err
	}


	// TODO: shutdown
	select {}

	return nil
}


func getClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

// Create TPR/CR if not already created, and custom Rest client
func createFuncCR(config *rest.Config, cl *kubernetes.Clientset) (*rest.RESTClient, error) {

	err := kubecr.FuncTPR.Create(cl)
	if err != nil {
		return nil, err
	}

	tprcl, _, err := kubecr.NewClient(config)
	if err != nil {
		return nil, err
	}

	// Wait for TPR to be ready (if it was just created)
	err = kubecr.FuncTPR.WaitForResource(tprcl)
	if err != nil {
		return nil, err
	}

	return tprcl, nil
}