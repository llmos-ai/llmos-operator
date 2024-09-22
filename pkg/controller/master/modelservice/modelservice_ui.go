package modelservice

import (
	"fmt"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/utils/reconcilehelper"
)

const (
	defaultGradioPort  = int32(7860)
	defaultGradioImage = "us-docker.pkg.dev/google-samples/containers/gke/gradio-app:v1.0.3"
)

func (h *handler) createModelServiceGUI(ms *mlv1.ModelService, ssService *corev1.Service) error {
	if !ms.Spec.EnableGUI {
		return nil
	}

	uiName := getFormattedMSName(ms.Name, "gradio")
	labels := getModelServiceGUILabels(ms.Name, uiName)
	deployment := constructGUIDeployment(ms, uiName, ssService, labels)

	foundDeployment, err := h.DeploymentCache.Get(deployment.Namespace, deployment.Name)
	if err != nil && errors.IsNotFound(err) {
		if _, err = h.Deployments.Create(deployment); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if foundDeployment != nil && reconcilehelper.CopyDeploymentFields(deployment, foundDeployment) {
		logrus.Debugf("updating deployment %s/%s", foundDeployment.Namespace, foundDeployment.Name)
		deploymentCopy := foundDeployment.DeepCopy()
		if _, err = h.Deployments.Update(deploymentCopy); err != nil {
			return err
		}
	}

	svc := constructGUIService(deployment, ms.Spec.ServiceType)
	foundSvc, err := h.ServiceCache.Get(svc.Namespace, svc.Name)
	if err != nil && errors.IsNotFound(err) {
		if _, err = h.Services.Create(svc); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if foundSvc != nil && reconcilehelper.CopyServiceFields(svc, foundSvc) {
		logrus.Debugf("updating model UI service %s/%s", foundSvc.Namespace, foundSvc.Name)
		toUpdate := foundSvc.DeepCopy()
		if _, err = h.Services.Update(toUpdate); err != nil {
			return err
		}
	}
	return nil
}

func constructGUIDeployment(ms *mlv1.ModelService, name string, svc *corev1.Service,
	labels map[string]string) *appsv1.Deployment {
	replicas := int32(1)
	servingUrl := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", svc.Name, svc.Namespace, svc.Spec.Ports[0].Port)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ms.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ms, mlv1.SchemeGroupVersion.WithKind(msKindName)),
			},
			Labels: labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: map[string]string{},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "gradio",
							Image: defaultGradioImage,
							Env: []corev1.EnvVar{
								{
									Name:  "MODEL_ID",
									Value: ms.Spec.ModelName,
								},
								{
									Name:  "CONTEXT_PATH",
									Value: "/v1/chat/completions",
								},
								{
									Name:  "HOST",
									Value: servingUrl,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: defaultGradioPort,
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
	}
}

func constructGUIService(deployment *appsv1.Deployment, serviceType corev1.ServiceType) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            deployment.Name,
			Namespace:       deployment.Namespace,
			OwnerReferences: deployment.OwnerReferences,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt32(defaultGradioPort),
				},
			},
			Selector: deployment.Spec.Template.Labels,
			Type:     serviceType,
		},
	}
}

func getModelServiceGUILabels(name, uiName string) map[string]string {
	return map[string]string{
		constant.LabelModelServiceName: name,
		constant.LabelLLMOSMLAppName:   uiName,
	}
}
