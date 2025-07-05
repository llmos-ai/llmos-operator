package datasetversion

import (
	"fmt"

	"github.com/llmos-ai/llmos-operator/pkg/webhook/config"
	"github.com/oneblock-ai/webhook/pkg/server/admission"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
)

type mutator struct {
	admission.DefaultMutator

	datasetCache ctlmlv1.DatasetCache
}

var _ admission.Mutator = &mutator{}

func NewMutator(mgmt *config.Management) admission.Mutator {
	return &mutator{
		datasetCache: mgmt.LLMFactory.Ml().V1().Dataset().Cache(),
	}
}

func (m *mutator) Create(request *admission.Request, newObj runtime.Object) (admission.Patch, error) {
	dv := newObj.(*mlv1.DatasetVersion)

	dataset, err := m.datasetCache.Get(dv.Namespace, dv.Spec.Dataset)
	if err != nil {
		return nil, fmt.Errorf("get dataset %s/%s failed: %v", dv.Namespace, dv.Spec.Dataset, err)
	}

	return []admission.PatchOp{
		addOwnerReference(dv.Spec.Dataset, "Dataset", dataset.UID),
		addLabels(dv.Spec.Dataset, dv.Spec.Version),
	}, nil
}

func (m *mutator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"datasetversions"},
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   mlv1.SchemeGroupVersion.Group,
		APIVersion: mlv1.SchemeGroupVersion.Version,
		ObjectType: &mlv1.DatasetVersion{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
		},
	}
}

func addOwnerReference(name, kind string, uid types.UID) admission.PatchOp {
	return admission.PatchOp{
		Op:   admission.PatchOpAdd,
		Path: "/metadata/ownerReferences",
		Value: []metav1.OwnerReference{
			{
				UID:        uid,
				APIVersion: mlv1.SchemeGroupVersion.String(),
				Kind:       kind,
				Name:       name,
			},
		},
	}
}

func addLabels(datasetName, datasetVersion string) admission.PatchOp {
	return admission.PatchOp{
		Op:   admission.PatchOpAdd,
		Path: "/metadata/labels",
		Value: map[string]string{
			constant.LabelDatasetName:    datasetName,
			constant.LabelDatasetVersion: datasetVersion,
		},
	}
}
