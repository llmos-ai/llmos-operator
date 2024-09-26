package modelservice

import (
	"github.com/oneblock-ai/webhook/pkg/server/admission"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/modelservice"
)

type mutator struct {
	admission.DefaultMutator
}

var _ admission.Mutator = &mutator{}

func NewMutator() admission.Mutator {
	return &mutator{}
}

func (m *mutator) Create(_ *admission.Request, newObj runtime.Object) (admission.Patch, error) {
	ms := newObj.(*mlv1.ModelService)

	patches := make([]admission.PatchOp, 0)

	patches = append(patches, patchSelector(ms))

	return patches, nil
}

func (m *mutator) Update(_ *admission.Request, _ runtime.Object, newObj runtime.Object) (admission.Patch, error) {
	ms := newObj.(*mlv1.ModelService)

	patches := make([]admission.PatchOp, 0)

	if ms.Spec.Selector == nil {
		patches = append(patches, patchSelector(ms))
	}

	return patches, nil
}

func patchSelector(ms *mlv1.ModelService) admission.PatchOp {
	return admission.PatchOp{
		Op:    admission.PatchOpReplace,
		Path:  "/spec/selector",
		Value: modelservice.GetModelServiceSelector(ms),
	}
}

func (m *mutator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"modelservices"},
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   mlv1.SchemeGroupVersion.Group,
		APIVersion: mlv1.SchemeGroupVersion.Version,
		ObjectType: &mlv1.ModelService{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}
