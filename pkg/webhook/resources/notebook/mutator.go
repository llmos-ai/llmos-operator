package notebook

import (
	"github.com/oneblock-ai/webhook/pkg/server/admission"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
)

type mutator struct {
	admission.DefaultMutator
}

var _ admission.Mutator = &mutator{}

func NewMutator() admission.Mutator {
	return &mutator{}
}

func (m *mutator) Create(_ *admission.Request, newObj runtime.Object) (admission.Patch, error) {
	notebook := newObj.(*mlv1.Notebook)

	patches := make([]admission.PatchOp, 0)

	rs := notebook.Spec.Template.Spec.Containers[0].Resources
	if rs.Requests != nil && rs.Limits == nil {
		op := patchResourceLimit(rs)
		patches = append(patches, op)
	}

	return patches, nil
}

func (m *mutator) Update(_ *admission.Request, _ runtime.Object, newObj runtime.Object) (admission.Patch, error) {
	notebook := newObj.(*mlv1.Notebook)

	patches := make([]admission.PatchOp, 0)

	rs := notebook.Spec.Template.Spec.Containers[0].Resources
	if rs.Requests != nil && rs.Limits == nil {
		op := patchResourceLimit(rs)
		patches = append(patches, op)
	}

	return patches, nil
}

func patchResourceLimit(rs corev1.ResourceRequirements) admission.PatchOp {
	return admission.PatchOp{
		Op:    admission.PatchOpReplace,
		Path:  "/spec/template/spec/containers/0/resources/limits",
		Value: rs.Requests,
	}
}

func (m *mutator) Resource() admission.Resource {
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
