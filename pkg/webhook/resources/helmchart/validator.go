package helmchart

import (
	"fmt"

	helmv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	"github.com/oneblock-ai/webhook/pkg/server/admission"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"

	werror "github.com/llmos-ai/llmos-operator/pkg/webhook/error"
)

type validator struct {
	admission.DefaultValidator
}

var _ admission.Validator = &validator{}

func NewValidator() admission.Validator {
	return &validator{}
}

func (v *validator) Delete(_ *admission.Request, obj runtime.Object) error {
	chart := obj.(*helmv1.HelmChart)
	if chart.Name == "llmos-crd" || chart.Name == "llmos-operator" {
		return werror.MethodNotAllowed(fmt.Sprintf("Can't delete LLMOS system chart %s", chart.Name))
	}
	return nil
}

func (v *validator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"helmcharts"},
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   helmv1.SchemeGroupVersion.Group,
		APIVersion: helmv1.SchemeGroupVersion.Version,
		ObjectType: &helmv1.HelmChart{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Delete,
		},
	}
}
