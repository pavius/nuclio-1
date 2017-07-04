package deploy

import (
	"strconv"
	"github.com/nuclio/nuclio/pkg/kubecr"
)

type FunctionStateRec struct {
	LastGen string
	LastVer int
	Alias string
}

type FuncCache struct {
	StateCache map[string]FunctionStateRec
}
var funcStateCache map[string]FunctionStateRec


func NewFuncCache() *FuncCache {
	fc := FuncCache{}
	fc.StateCache = make(map[string]FunctionStateRec)
	return &fc
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
	if err != nil {return false}  // TODO: need to log
	li, err := strconv.Atoi(fc.StateCache[namespace + "." + name].LastGen)
	if err != nil {return false}  // TODO: need to log
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
	if val, ok := funcStateCache[f.Namespace + "." + f.Name]; ok {
		val.LastVer = f.Spec.Version
		val.Alias = f.Spec.Alias
		fc.StateCache[f.Namespace + "." + f.Name] = val
	} else {
		state := FunctionStateRec{ LastVer:f.Spec.Version, Alias:f.Spec.Alias}
		funcStateCache[f.Namespace + "." + f.Name] = state
	}
}
