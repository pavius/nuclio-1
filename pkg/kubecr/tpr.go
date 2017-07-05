package kubecr

import (
	"k8s.io/client-go/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1b1e "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

const (
	TPRName        string = "function"
	TPRPlural      string = "functions"
	TPRGroup       string = "nuclio.io"
	TPRVersion     string = "v1"
	TPRDescription string = "My TPR"
)

type tpr struct {
	Name string
	Plural string
	Description string
	KnownType func(scheme *runtime.Scheme) error
}

func (t tpr) Create(clientset kubernetes.Interface) error {
	tpr:= &v1b1e.ThirdPartyResource{
		ObjectMeta: meta_v1.ObjectMeta{Name:t.Name + "." + TPRGroup},
		Versions: []v1b1e.APIVersion{{Name:TPRVersion}},
		Description: t.Description,
	}

	_, err := clientset.Extensions().ThirdPartyResources().Create(tpr)
	if err != nil && apierrors.IsAlreadyExists(err) {
		return nil
	}
	if err == nil {
		time.Sleep(time.Second)  // if created wait for the TPR to update
	}
	return err
}


func (t tpr) WaitForResource(cl *rest.RESTClient) error {
	return wait.Poll(100*time.Millisecond, 60*time.Second, func() (bool, error) {
		_, err := cl.Get().Namespace(v1.NamespaceDefault).Resource(t.Plural).DoRaw()
		if err == nil {
			return true, nil
		}
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	})
}

var SchemeGroupVersion = schema.GroupVersion{Group: TPRGroup, Version: TPRVersion}

func NewClient(cfg *rest.Config) (*rest.RESTClient, *runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	SchemeBuilder := runtime.NewSchemeBuilder(FuncTPR.KnownType)
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, nil, err
	}
	config := *cfg
	config.GroupVersion = &SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}
	return client, scheme, nil
}


func GetClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

// Create TPR/CR if not already created, and custom Rest client
func CreateFuncCR(config *rest.Config, cl *kubernetes.Clientset) (*rest.RESTClient, error) {

	err := FuncTPR.Create(cl)
	if err != nil {
		return nil, err
	}

	tprcl, _, err := NewClient(config)
	if err != nil {
		return nil, err
	}

	// Wait for TPR to be ready (if it was just created)
	err = FuncTPR.WaitForResource(tprcl)
	if err != nil {
		return nil, err
	}

	return tprcl, nil
}