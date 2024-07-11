package notebook

import (
	"github.com/oneblock-ai/webhook/pkg/server/admission"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"

	mlv1 "github.com/llmos-ai/llmos-controller/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-controller/pkg/constant"
	"github.com/llmos-ai/llmos-controller/pkg/utils"
)

type validator struct {
	admission.DefaultValidator
}

var _ admission.Validator = &validator{}

func NewValidator() admission.Validator {
	return &validator{}
}

func (v *validator) Create(_ *admission.Request, newObj runtime.Object) error {
	notebook := newObj.(*mlv1.Notebook)

	return validateVolumeClaimTemplatesAnnotation(notebook)
}

func (v *validator) Update(_ *admission.Request, _, newObj runtime.Object) error {
	notebook := newObj.(*mlv1.Notebook)

	return validateVolumeClaimTemplatesAnnotation(notebook)
}

func validateVolumeClaimTemplatesAnnotation(cluster *mlv1.Notebook) error {
	volumeClaimTemplates, ok := cluster.Annotations[constant.AnnotationVolumeClaimTemplates]
	if !ok || volumeClaimTemplates == "" {
		return nil
	}
	return utils.ValidateVolumeClaimTemplatesAnnotation(volumeClaimTemplates)
}

func (v *validator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"notebooks"},
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   mlv1.SchemeGroupVersion.Group,
		APIVersion: mlv1.SchemeGroupVersion.Version,
		ObjectType: &mlv1.Notebook{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}
