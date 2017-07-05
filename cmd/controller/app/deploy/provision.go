package deploy

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	autos_v1 "k8s.io/client-go/pkg/apis/autoscaling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/pkg/apis/apps/v1beta1"
	"github.com/nuclio/nuclio/pkg/util/common"
	"github.com/nuclio/nuclio/pkg/kubecr"
)


const (
	SERVERLESS_PFX = "NUCLIO_"   // vs AWS_LAMBDA_
	PLATFORM_PFX  = "IGZ_"       // vs AWS_
	SERVERLESS_LABEL = "nuclio"  // for Kubernetes
	MAX_AUTOSCALE_PODS = 4
)

func UpdateContainer(f *kubecr.Function, lbl map[string]string, cont *v1.Container) {
	// fill the environment vaiables like AWS Lambda
	env := GetFuncEnv(f,lbl)

	cont.Image = f.Spec.Image
	cont.Ports = []v1.ContainerPort{{ContainerPort:80}}
	cont.Resources = f.Spec.Resources
	cont.WorkingDir = f.Spec.WorkingDir
	cont.Env = env

	// TBD: relevant volume mounts
	// TBD: Command, Args, LivenessProbe, Lifecycle

}

// Create or Update the Kubernetes Service
func ProvisionService(cl *kubernetes.Clientset, f *kubecr.Function, lbl map[string]string) (error) {
	svc, err := cl.Core().Services(f.Namespace).Get(f.Name, meta_v1.GetOptions{})
	if apierrors.IsNotFound(err) {
		svc, err := cl.Core().Services(f.Namespace).Create(&v1.Service{
			ObjectMeta: meta_v1.ObjectMeta{Name:f.Name, Namespace:f.Namespace, Labels:lbl},
			Spec: v1.ServiceSpec{Ports: []v1.ServicePort{{Name:"web",Port: int32(80)}},
				Selector: lbl,
				Type: "NodePort",
			},
		})
		if err != nil {
			common.LogDebug("Created Service: \n %+v",svc)
		}
		return err
	} else if err != nil {
		return err
	}

	svc.Labels = lbl
	svc.Spec.Ports = []v1.ServicePort{{Name:"web",Port: int32(80)}}
	svc.Spec.Selector = lbl
	svc.Spec.Type = "NodePort"

	// TODO: handle  apierrors.IsConflict  retry get+modify+update
	common.LogDebug("Update Service: \n %+v",svc)
	_, err = cl.Core().Services(f.Namespace).Update(svc)

	return err

}

// Create or Update the Kubernetes Deployment
func ProvisionDeployment(cl *kubernetes.Clientset, f *kubecr.Function, lbl map[string]string) (error) {
	replicas := f.Spec.Replicas
	if f.Spec.Disabled {
		replicas = 0
	} else if replicas == 0 {
		replicas = f.Spec.MinReplicas
	}

	annotations, err := GetFuncAnnot(f)
	if err != nil {
		return err
	}

	dep, err := cl.AppsV1beta1().Deployments(f.Namespace).Get(f.Name, meta_v1.GetOptions{})
	if apierrors.IsNotFound(err) {

		cont := v1.Container{Name: "nuclio"}
		UpdateContainer(f, lbl, &cont)

		pod := v1.PodTemplateSpec{
			meta_v1.ObjectMeta{Name: f.Name, Namespace: f.Namespace, Labels:lbl},
			v1.PodSpec{Containers:[]v1.Container{cont}},
		}
		common.LogDebug("POD: \n %+v",pod)

		dep, err := cl.AppsV1beta1().Deployments(f.Namespace).Create(&v1beta1.Deployment{
			ObjectMeta: meta_v1.ObjectMeta{Name:f.Name, Namespace:pod.Namespace, Labels:lbl , Annotations:annotations},
			Spec:v1beta1.DeploymentSpec{Replicas:&replicas, Template:pod},
		})
		if err != nil {
			common.LogDebug("Created Deployment: \n %+v",dep)
		}
		return err
	} else if err != nil {
		return err
	}

	dep.Labels = lbl
	dep.Annotations = annotations
	dep.Spec.Replicas = &replicas
	dep.Spec.Template.Labels = lbl
	UpdateContainer(f, lbl, &dep.Spec.Template.Spec.Containers[0])

	// TODO: handle  apierrors.IsConflict  retry get+modify+update
	common.LogDebug("Update Deployment: \n %+v",dep)
	_, err = cl.AppsV1beta1().Deployments(f.Namespace).Update(dep)

	return err
}

// Create or Update Function Resources (Service, Deployment, Auto-scaler, TBD: ConfigMap for events)
func DeployFunction(cl *kubernetes.Clientset, f *kubecr.Function) (error) {

	lbl, err := GetFuncLabels(f)
	if err != nil {
		return err
	}

	// Create or Update a Kubernetes service
	err = ProvisionService(cl,f,lbl)
	if err != nil {
		return err
	}

	// TODO: we may need to create a ConfigMap or map one to Deployment for event sources

	// Create a Kubernetes deployment
	err = ProvisionDeployment(cl,f,lbl)
	if err != nil {
		return err
	}

	// Create a Kubernetes horizontal pod autoscaler if Replica number is not specified
	var hpa *autos_v1.HorizontalPodAutoscaler
	hpa, err = cl.Autoscaling().HorizontalPodAutoscalers(f.Namespace).Get(f.Name,meta_v1.GetOptions{})

	if f.Spec.Replicas == 0 && f.Spec.Disabled != true {
		var max int32 = MAX_AUTOSCALE_PODS
		if f.Spec.MaxReplicas != 0 {
			max = f.Spec.MaxReplicas
		}
		min := f.Spec.MinReplicas
		if min == 0 {
			min = 1
		}
		var targetCPU int32 = 80

		if apierrors.IsNotFound(err) {
			// create new HPA
			hpa, err = cl.Autoscaling().HorizontalPodAutoscalers(f.Namespace).Create(&autos_v1.HorizontalPodAutoscaler{
				ObjectMeta: meta_v1.ObjectMeta{Name:f.Name, Namespace:f.Namespace, Labels:lbl},
				Spec: autos_v1.HorizontalPodAutoscalerSpec{
					MinReplicas: &min,
					MaxReplicas: max,
					TargetCPUUtilizationPercentage: &targetCPU,
					ScaleTargetRef: autos_v1.CrossVersionObjectReference{
						APIVersion: "extensions/v1beta1",
						Kind:"Deployment",
						Name:f.Name,
					},

				},
			})
			if err != nil {
				return err
			}
		} else {
			// Update existing HPA
			hpa.Labels = lbl
			hpa.Spec.MinReplicas = &min
			hpa.Spec.MaxReplicas = max
			hpa.Spec.TargetCPUUtilizationPercentage = &targetCPU
			_ , err = cl.Autoscaling().HorizontalPodAutoscalers(f.Namespace).Update(hpa)
		}
		common.LogDebug("HPA: \n %+v",hpa)
	} else if err == nil {
		// we dont need the HPA , can delete if it exists
		err = cl.Autoscaling().HorizontalPodAutoscalers(f.Namespace).Delete(f.Name, &meta_v1.DeleteOptions{})
		if err != nil {
			common.LogDebug("Cant Delete HPA: %s %s - %s",f.Namespace,f.Name,err)
		}
	}
	return nil
}

// Delete Function Resources - Service, Deployment, Auto-Scale  (if they exist)
func DeleteFunc(cl *kubernetes.Clientset, namespace, name string) error {
	prop := meta_v1.DeletePropagationForeground
	delopt := &meta_v1.DeleteOptions{PropagationPolicy: &prop}

	// Delete Auto Scaler if exists
	err := cl.Autoscaling().HorizontalPodAutoscalers(namespace).Delete(name, delopt)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	} else if !apierrors.IsNotFound(err){
		common.LogDebug("Deleted HPA: %s %s",namespace, name)
	}

	// Delete Service if exists
	err = cl.Core().Services(namespace).Delete(name, delopt)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	} else if !apierrors.IsNotFound(err){
		common.LogDebug("Deleted Service: %s.%s",namespace, name)
	}

	// Delete Deployment if exists
	err = cl.AppsV1beta1().Deployments(namespace).Delete(name, delopt)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	} else if !apierrors.IsNotFound(err){
		common.LogDebug("Deleted Deployment: %s %s",namespace, name)
	}

	return nil
}
