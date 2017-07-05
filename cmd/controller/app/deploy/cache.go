package deploy

import (
	"strconv"
	"github.com/nuclio/nuclio/pkg/kubecr"
	"fmt"
	"k8s.io/client-go/kubernetes"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/nuclio/nuclio/pkg/util/common"
)

type FunctionStateRec struct {
	LastGen string
	LastVer int
	Alias string
}

type FuncCache struct {
	StateCache map[string]FunctionStateRec
	cl *kubernetes.Clientset
}


func NewFuncCache(cl *kubernetes.Clientset) *FuncCache {
	fc := FuncCache{cl:cl}
	fc.StateCache = make(map[string]FunctionStateRec)
	return &fc
}

func (fc *FuncCache) Init(namespace string) error {
	opts := meta_v1.ListOptions{
		LabelSelector: "serverless="+SERVERLESS_LABEL,
	}
	deps, err := fc.cl.AppsV1beta1().Deployments("").List(opts)
	if err != nil {
		return err
	}
	fmt.Printf("deps: %s \n",deps)

	for _,dep := range deps.Items {
		gen   := dep.Annotations["func_gen"]
		line  := FunctionStateRec{LastGen:gen, Alias:dep.Labels["alias"]}
		fc.Set(dep.Namespace,dep.Name,line)
	}
	return nil
}

func (fc *FuncCache) Set(namespace, name string, state FunctionStateRec) {
	fc.StateCache[namespace + "." + name] = state
}

func (fc *FuncCache) Get(namespace, name string) FunctionStateRec {
	return fc.StateCache[namespace + "." + name]
}

func (fc *FuncCache) Del(namespace, name string)  {
	delete(fc.StateCache, namespace + "." + name)
}

func (fc *FuncCache) DidStateChange(namespace, name, gen string) bool {
	gi, err := strconv.Atoi(gen)
	if err != nil {
		common.LogDebug("failed to convert gen to int: %s %s (%s)",namespace, name, gen)
		return false
	}

	if _, ok := fc.StateCache[namespace + "." + name]; !ok {
		return true
	}
	li, err := strconv.Atoi(fc.StateCache[namespace + "." + name].LastGen)
	if err != nil {
		common.LogDebug("failed to convert last to int: %s %s (%s)",namespace, name, gen)
		return false
	}
	common.LogDebug("Did Change? Gen: %d - Last: %d ",gi,li)
	// New update is more recent than the last gen we saw
	return gi > li
}

func (fc *FuncCache) SetGen(namespace, name, gen string) {
	if val, ok := fc.StateCache[namespace + "." + name]; ok {
		val.LastGen = gen
		fc.StateCache[namespace + "." + name] = val
	} else {
		state := FunctionStateRec{ LastGen:gen}
		fc.StateCache[namespace + "." + name] = state
	}
}

func (fc *FuncCache) SetArgs(f *kubecr.Function) {
	if val, ok := fc.StateCache[f.Namespace + "." + f.Name]; ok {
		val.LastVer = f.Spec.Version
		val.Alias = f.Spec.Alias
		fc.StateCache[f.Namespace + "." + f.Name] = val
	} else {
		state := FunctionStateRec{ LastVer:f.Spec.Version, Alias:f.Spec.Alias}
		fc.StateCache[f.Namespace + "." + f.Name] = state
	}
}
