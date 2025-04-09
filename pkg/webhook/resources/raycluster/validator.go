package raycluster

import (
	"fmt"

	"github.com/oneblock-ai/webhook/pkg/server/admission"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	"github.com/sirupsen/logrus"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlrayv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ray.io/v1"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/config"
)

type validator struct {
	admission.DefaultValidator
	rayClusterCache ctlrayv1.RayClusterCache
}

var _ admission.Validator = &validator{}

func NewValidator(mgmt *config.Management) admission.Validator {
	return &validator{
		rayClusterCache: mgmt.RayFactory.Ray().V1().RayCluster().Cache(),
	}
}

func (v *validator) Create(_ *admission.Request, newObj runtime.Object) error {
	cluster := newObj.(*rayv1.RayCluster)

	logrus.Debugf("raycluster %s is created", cluster.Name)

	if err := validateAutoScalingWithWorkerGroupSpecs(cluster); err != nil {
		return err
	}

	return validateVolumeClaimTemplatesAnnotation(cluster)
}

func (v *validator) Update(_ *admission.Request, _, newObj runtime.Object) error {
	cluster := newObj.(*rayv1.RayCluster)

	logrus.Debugf("raycluster %s is updated", cluster.Name)

	if err := validateAutoScalingWithWorkerGroupSpecs(cluster); err != nil {
		return err
	}

	return validateVolumeClaimTemplatesAnnotation(cluster)
}

// validateAutoScalingWithWorkerGroupSpecs checks if enableInTreeAutoscaling is true, workerGroupSpecs should be defined
func validateAutoScalingWithWorkerGroupSpecs(cluster *rayv1.RayCluster) error {
	if cluster.Spec.EnableInTreeAutoscaling != nil && *cluster.Spec.EnableInTreeAutoscaling {
		if len(cluster.Spec.WorkerGroupSpecs) == 0 {
			return fmt.Errorf("enableInTreeAutoscaling is true, but workerGroupSpecs is not defined")
		}
	}
	return nil
}

func validateVolumeClaimTemplatesAnnotation(cluster *rayv1.RayCluster) error {
	volumeClaimTemplates, ok := cluster.Annotations[constant.AnnotationVolumeClaimTemplates]
	if !ok || volumeClaimTemplates == "" {
		return nil
	}
	return utils.ValidateVolumeClaimTemplatesAnnotation(volumeClaimTemplates)
}

func (v *validator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"rayclusters"},
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   rayv1.SchemeGroupVersion.Group,
		APIVersion: rayv1.SchemeGroupVersion.Version,
		ObjectType: &rayv1.RayCluster{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}
