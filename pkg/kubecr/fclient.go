package kubecr

import (
	"k8s.io/client-go/rest"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"time"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
)

func Functions(cl *rest.RESTClient) *functions {
	return &functions{cl: cl, plural:TPRPlural}
}

type functions struct {
	cl *rest.RESTClient
	plural string
}

func (f *functions) Create(obj *Function) (*Function,error) {
	var result Function
	err := f.cl.Post().
		Namespace(obj.ObjectMeta.Namespace).Resource(f.plural).
		Body(obj).Do().Into(&result)
	return &result, err
}

func (f *functions) Update(obj *Function) (*Function,error) {
	var result Function
	err := f.cl.Put().
		Namespace(obj.ObjectMeta.Namespace).Name(obj.ObjectMeta.Name).Resource(f.plural).
		Body(obj).Do().Into(&result)
	return &result, err
}


func (f *functions) Delete(namespace, name string, options *meta_v1.DeleteOptions) error {
	return f.cl.Delete().
		Namespace(namespace).Resource(f.plural).
		Name(name).Body(options).Do().
		Error()
}


func (f *functions) Get(namespace, name string) (*Function,error) {
	var result Function
	err := f.cl.Get().
		Namespace(namespace).Resource(f.plural).
		Name(name).Do().Into(&result)
	return &result, err
}


func (f *functions) List(namespace string) (*FunctionList,error) {
	var result FunctionList
	err := f.cl.Get().
		Namespace(namespace).Resource(f.plural).
		Do().Into(&result)
	return &result, err
}



func (f *functions) NewListWatch(namespace string) *cache.ListWatch {
	return cache.NewListWatchFromClient(f.cl,f.plural,namespace,fields.Everything())
}

func (f *functions) WaitForProcessed(namespace, name string) error {
	return wait.Poll(100*time.Millisecond, 10*time.Second, func() (bool, error) {
		function, err := f.Get(namespace, name)
		if err == nil && function.Status.State != FunctionStateCreated {
			return true, nil
		}
		return false, err
	})
}