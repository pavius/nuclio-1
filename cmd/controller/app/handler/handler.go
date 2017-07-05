package handler

import (
	"github.com/nuclio/nuclio/cmd/controller/app/deploy"
	"fmt"
	"strings"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"github.com/nuclio/nuclio/pkg/kubecr"
	"github.com/nuclio/nuclio/pkg/util/common"
	"github.com/nuclio/nuclio/pkg/logger"
)




type Handler struct {
	logger       logger.Logger
	cs           *kubernetes.Clientset
	crcl         *rest.RESTClient
	stateCache   *deploy.FuncCache
}

func NewHandler(cs *kubernetes.Clientset, crcl *rest.RESTClient, cache *deploy.FuncCache) (*Handler, error) {

	newHandler := Handler{ cs:cs, crcl:crcl, stateCache:cache }

	return &newHandler, nil
}


func (h *Handler) HandleFuncChange(namespace, name, gen string) error {

	if namespace == "" {
		return fmt.Errorf("ERROR null namespace with %s\n",name)
	}

	// TODO: change to IsNewerState
	if !h.stateCache.DidStateChange(namespace, name, gen) {
		common.LogDebug("Add/Update with no change to: %s %s (%s) \n",namespace, name, gen)
		return nil
	}

	fapi := kubecr.Functions(h.crcl)
	function, err := fapi.Get(namespace, name)
	if err != nil {
		common.LogDebug("Error with function get: %s %s (%s)\n",namespace, name, err)
		// TODO if unmarshal error may need to mark function w error
		return err
	}
	fname:=function.Name
	common.LogDebug("HandleFuncChange-Get: %s %s (G %s , RV %s, C %s) \n",namespace, name, gen, function.ResourceVersion, h.stateCache.Get(namespace, name).LastGen)



	if function.ObjectMeta.Labels == nil {
		function.ObjectMeta.Labels = make(map[string]string)
	}

	// extract function version number from the name (the number after the '-' if not latest)
	ver := 0
	if idx := strings.LastIndex(name, "-"); idx>0 {
		v, err := strconv.Atoi(name[idx+1:len(name)])
		if err == nil && v>0 && function.Spec.Version>0  && function.Labels["function"]==name[0:idx] {
			ver = v
			fname = name[0:idx]
		}
	}

	//return updateFunc(cl, function, kubecr.FunctionStateError,
	//	"Error!, Cannot use Dot in a function name")

	if function.Labels["function"] !="" && fname != function.Labels["function"] {
		return h.updateFunc(function, kubecr.FunctionStateError,
			"Error!, Name and function lable must be the same")
	}
	function.ObjectMeta.Labels["function"]=fname

	// verify the ver num from the name match the spec ver num (i.e. wasnt modified)
	// TODO: move all new ver tests to function, add test that image & code are noth both null etc.
	// TODO: move all old ver tests to function, add test that image is not null etc.
	if ver > 0 && function.Spec.Version != ver {
		return h.updateFunc(function, kubecr.FunctionStateError,
			"Error!, version number cannot be modified on published versions ")
	}

	if ver > 0 && function.Spec.Alias == "latest" {
		return h.updateFunc(function, kubecr.FunctionStateError,
			"Error!, Older versions cannot be tagged as 'latest' ")
	}

	if function.Spec.Version == 0 {
		// New Function
		function.Spec.Version = 1
		function.Spec.Alias = "latest"
	}

	if ver == 0 && function.Spec.Alias != "latest" && !function.Spec.Publish {
		return h.updateFunc(function, kubecr.FunctionStateError,
			"Error!, Head version must be tagged as 'latest' or use Publish flag")
	}

	if function.Spec.Alias == "latest" {
		function.ObjectMeta.Labels["version"] = "latest"
	} else {
		function.ObjectMeta.Labels["version"] = strconv.Itoa(function.Spec.Version)
	}

	if function.Spec.Image == "" || function.Spec.Disabled {

		if function.Spec.Publish {
			return h.updateFunc(function, kubecr.FunctionStateError,
				"Error!, Can't Publish on build or disabled function")
		}

		msg := ""
		if function.Spec.Image == "" &&
			(function.Status.BuildState == kubecr.BuildStateReady ||
				function.Status.BuildState == kubecr.BuildStateUnknown) {
			function.Status.BuildState = kubecr.BuildStatePending
			msg = "Build pending"
		}
		err = h.updateFunc(function, kubecr.FunctionStateProcessed, msg)
		if err != nil {
			return err
		}
		h.stateCache.SetArgs(function)
		if function.Spec.Disabled {
			return deploy.DeleteFunc(h.cs, namespace, name)
		}

		return nil
	}


	// TODO: if alias !="" and changed need to check which versions had the old tag (if any)

	if function.Spec.Publish {

		// TODO: if alias = "latest" clear it, check code above for no conflicts

		if ver != 0 {
			return h.updateFunc(function, kubecr.FunctionStateError,
				"Error!, Cannot publish version other than latest")
		}

		pubver := function.Spec.Version
		function.Spec.Version += 1
		function.Spec.Publish = false

		err = h.updateFunc(function, kubecr.FunctionStateProcessed, "")
		if err != nil {
			return err
		}
		common.LogDebug("Updated Func before publish")

		lbl := function.Labels
		lbl["alias"] = function.Spec.Alias
		lbl["version"] = strconv.Itoa(pubver)

		function.ObjectMeta = meta_v1.ObjectMeta{
			Name: function.Name+"-"+lbl["version"],
			Namespace:function.Namespace,
			Labels: lbl,
		}
		function.Spec.Version = pubver
		err = h.createFunc(function, kubecr.FunctionStateProcessed, "")
		if err != nil {
			common.LogDebug("Failed to create new published version in %s %s - %s",namespace, name, err)
			return err
		}
	} else if ver != 0 {

		// TODO: if alias changed need to check which versions had the old tag (if any)
		function.ObjectMeta.Labels["alias"] = function.Spec.Alias

	}

	common.Print_json(function)
	err = h.updateFunc(function, kubecr.FunctionStateProcessed, "")
	if err != nil {
		return err
	}
	common.LogDebug("Updated Func S3")

	// create or update
	err = deploy.DeployFunction(h.cs, function)
	if err != nil {
		return err
	}

	// TODO: delete alias from old if needed


	return nil
}

func (h *Handler) updateFunc(f *kubecr.Function, state kubecr.FunctionState, msg string) error {
	f.Status.ObservedGen = f.ResourceVersion
	f.Status.State = state
	f.Status.Message = msg

	common.LogDebug("Updating function status: %s %s (%s) \n",f.Namespace, f.Name, msg)
	newf, err := kubecr.Functions(h.crcl).Update(f)
	if err != nil {
		// TODO handle different err types, e.g. res ver conflict
		return err
	}
	h.stateCache.SetGen(f.Namespace, f.Name, newf.ResourceVersion)
	return nil
}

func (h *Handler) createFunc(f *kubecr.Function, state kubecr.FunctionState, msg string) error {
	f.Status.ObservedGen = "0"
	f.Status.State = state
	f.Status.Message = msg

	common.LogDebug("Create new function: %s %s (%s) \n",f.Namespace, f.Name, msg)
	newf, err := kubecr.Functions(h.crcl).Create(f)
	if err != nil {
		// TODO handle different err types, e.g. already exist
		return err
	}
	h.stateCache.SetGen(f.Namespace, f.Name, newf.ResourceVersion)
	return nil
}


