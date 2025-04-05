package dataset

import (
	"fmt"

	"github.com/oneblock-ai/webhook/pkg/server/admission"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/config"
	werror "github.com/llmos-ai/llmos-operator/pkg/webhook/error"
)

type validator struct {
	admission.DefaultValidator
	registryCache ctlmlv1.RegistryCache
}

var _ admission.Validator = &validator{}

func NewValidator(mgmt *config.Management) admission.Validator {
	return &validator{
		registryCache: mgmt.LLMFactory.Ml().V1().Registry().Cache(),
	}
}

func (v *validator) Create(_ *admission.Request, obj runtime.Object) error {
	d := obj.(*mlv1.Dataset)

	if _, err := v.registryCache.Get(d.Spec.Registry); err != nil {
		if errors.IsNotFound(err) {
			return werror.BadRequest(fmt.Sprintf("registry %s not found", d.Spec.Registry))
		}
		return werror.InternalError(fmt.Sprintf("get registry %s failed: %v", d.Spec.Registry, err))
	}

	return nil
}

func (v *validator) Update(_ *admission.Request, oldObj runtime.Object, newObj runtime.Object) error {
	oldD := oldObj.(*mlv1.Dataset)
	newD := newObj.(*mlv1.Dataset)

	if oldD.Spec.Registry != newD.Spec.Registry {
		return werror.MethodNotAllowed("registry field cannot be modified once set")
	}

	return nil
}

func (v *validator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"datasets"},
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   mlv1.SchemeGroupVersion.Group,
		APIVersion: mlv1.SchemeGroupVersion.Version,
		ObjectType: &mlv1.Dataset{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}
