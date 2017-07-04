package deploy

import (
	"bytes"
	"github.com/nuclio/nuclio/pkg/kubecr"
	"k8s.io/client-go/pkg/api/v1"
	"encoding/json"
)

func GetFuncJson(f *kubecr.Function) (string, error) {
	// store function spec as annotation
	body, err := json.Marshal(f.Spec)
	if err != nil {
		return "", err
	}
	var pbody bytes.Buffer
	err = json.Compact(&pbody, body)
	return string(pbody.Bytes()), err
}

func GetFuncLabels(f *kubecr.Function) (map[string]string,error) {
	labels := make(map[string]string)
	for k,v := range f.Labels {
		labels[k] = v
	}
	labels["serverless"] = SERVERLESS_LABEL

	return labels, nil
}

func GetFuncAnnot(f *kubecr.Function) (map[string]string,error) {
	annotations := make(map[string]string)
	if f.Spec.Description !="" {
		annotations["description"] = f.Spec.Description
	}
	js, err := GetFuncJson(f)
	if err != nil {
		return annotations, err
	}
	annotations["func_json"] = js
	annotations["func_gen"] = f.ResourceVersion
	return annotations, nil
}

func GetFuncEnv(f *kubecr.Function, lbl map[string]string) []v1.EnvVar {
	env := f.Spec.Env
	addEnv(&env, SERVERLESS_PFX + "REGION", "local")
	addEnv(&env, SERVERLESS_PFX + "LOG_STREAM_NAME", "local")
	addEnv(&env, SERVERLESS_PFX + "DLQ_STREAM_NAME", "")
	addEnv(&env, SERVERLESS_PFX + "FUNCTION_NAME", lbl["function"])
	addEnv(&env, SERVERLESS_PFX + "FUNCTION_VERSION", lbl["version"])
	addEnv(&env, SERVERLESS_PFX + "FUNCTION_MEMORY_SIZE", "TBD")

	addEnv(&env, PLATFORM_PFX + "ACCESS_KEY", "TBD")
	addEnv(&env, PLATFORM_PFX + "SECRET__KEY", "TBD")
	addEnv(&env, PLATFORM_PFX + "SESSION_TOKEN", "TBD")
	addEnv(&env, PLATFORM_PFX + "SECURITY_TOKEN", "TBD")

	//addEnv(&env, "NODE_PATH", "TBD")
	//addEnv(&env, "PYTHON_PATH", "TBD")

	return env
}


func addEnv(env *[]v1.EnvVar, key, val string) {
	*env = append(*env, v1.EnvVar{Name:key, Value:val} )
}


