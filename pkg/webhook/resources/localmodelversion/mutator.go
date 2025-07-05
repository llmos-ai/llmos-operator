package localmodelversion

import (
	"fmt"
	"strings"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/config"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/oneblock-ai/webhook/pkg/server/admission"
)

type mutator struct {
	admission.DefaultMutator

	localModelCache ctlmlv1.LocalModelCache
}

var _ admission.Mutator = &mutator{}

func NewMutator(mgmt *config.Management) admission.Mutator {
	return &mutator{
		localModelCache: mgmt.LLMFactory.Ml().V1().LocalModel().Cache(),
	}
}

func (m *mutator) Create(request *admission.Request, newObj runtime.Object) (admission.Patch, error) {
	lmv := newObj.(*mlv1.LocalModelVersion)
	localModel, err := m.localModelCache.Get(lmv.Namespace, lmv.Spec.LocalModel)
	if err != nil {
		return nil, fmt.Errorf("failed to get local model %s/%s: %w", lmv.Namespace, lmv.Spec.LocalModel, err)
	}

	tmp := strings.Split(localModel.Spec.ModelName, "/")
	if len(tmp) != 2 {
		return nil, fmt.Errorf("invalid model name %s of local model %s/%s",
			localModel.Spec.ModelName, localModel.Namespace, localModel.Name)
	}

	return admission.Patch{
		addLabels(localModel, tmp[0], tmp[1]),
		addOwnerReference(localModel),
	}, nil
}

func (m *mutator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"localmodelversions"},
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   mlv1.SchemeGroupVersion.Group,
		APIVersion: mlv1.SchemeGroupVersion.Version,
		ObjectType: &mlv1.LocalModelVersion{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
		},
	}
}

func addLabels(lm *mlv1.LocalModel, modelNamespace, modelName string) admission.PatchOp {
	return admission.PatchOp{
		Op:   admission.PatchOpAdd,
		Path: "/metadata/labels",
		Value: map[string]string{
			constant.LabelLocalModelName: lm.Name,
			constant.LabelRegistryName:   lm.Spec.Registry,
			constant.LabelModelName:      modelName,
			constant.LabelModelNamespace: modelNamespace,
		},
	}
}

func addOwnerReference(localModel *mlv1.LocalModel) admission.PatchOp {
	return admission.PatchOp{
		Op:   admission.PatchOpAdd,
		Path: "/metadata/ownerReferences",
		Value: []metav1.OwnerReference{
			{
				UID:        localModel.UID,
				APIVersion: mlv1.SchemeGroupVersion.String(),
				Kind:       localModel.Kind,
				Name:       localModel.Name,
			},
		},
	}
}
