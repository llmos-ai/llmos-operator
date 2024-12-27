package namespace

import (
	"fmt"

	"github.com/oneblock-ai/webhook/pkg/server/admission"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/data"
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
	ns := obj.(*corev1.Namespace)

	if !canDelete(ns) {
		return werror.MethodNotAllowed(fmt.Sprintf("Can't delete LLMOS reserved system namespace %s", ns.Name))
	}

	return nil
}

func canDelete(namespace *corev1.Namespace) bool {
	if namespace.Labels != nil && namespace.Labels[constant.AnnotationSkipWebhook] == "true" {
		return true
	}

	for _, ns := range data.ReservedSystemNamespaces {
		if ns == namespace.Name {
			return false
		}
	}

	return true
}

func (v *validator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"namespaces"},
		Scope:      admissionregv1.ClusterScope,
		APIGroup:   corev1.SchemeGroupVersion.Group,
		APIVersion: corev1.SchemeGroupVersion.Version,
		ObjectType: &corev1.Namespace{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Delete,
		},
	}
}
