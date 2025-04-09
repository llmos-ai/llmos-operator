package datasetversion

import (
	"fmt"
	"reflect"

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

	datasetCache ctlmlv1.DatasetCache
}

var _ admission.Validator = &validator{}

func Newvalidator(mgmt *config.Management) admission.Validator {
	return &validator{
		datasetCache: mgmt.LLMFactory.Ml().V1().Dataset().Cache(),
	}
}

func (v *validator) Create(_ *admission.Request, obj runtime.Object) error {
	dv := obj.(*mlv1.DatasetVersion)

	ds, err := v.datasetCache.Get(dv.Namespace, dv.Spec.Dataset)
	if err != nil {
		if errors.IsNotFound(err) {
			return werror.BadRequest(fmt.Sprintf("dataset %s/%s not found", dv.Namespace, dv.Spec.Dataset))
		}
		return werror.InternalError(fmt.Sprintf("get dataset %s/%s failed: %v", dv.Namespace, dv.Name, err))
	}

	// version should be unique in the dataset
	for _, v := range ds.Status.Versions {
		if v.Version == dv.Spec.Version {
			return werror.BadRequest(fmt.Sprintf("dataset %s/%s already has version %s",
				dv.Namespace, dv.Spec.Dataset, dv.Spec.Version))
		}
	}

	return nil
}

func (v *validator) Update(_ *admission.Request, oldObj runtime.Object, newObj runtime.Object) error {
	oldDV := oldObj.(*mlv1.DatasetVersion)
	newDV := newObj.(*mlv1.DatasetVersion)

	if oldDV.Spec.Dataset != newDV.Spec.Dataset || oldDV.Spec.Version != newDV.Spec.Version {
		return werror.MethodNotAllowed("dataset and version field cannot be modified once set")
	}

	if !reflect.DeepEqual(oldDV.Spec.CopyFrom, newDV.Spec.CopyFrom) {
		return werror.MethodNotAllowed("copyFrom field cannot be modified once set")
	}

	return nil
}

func (v *validator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"datasetversions"},
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   mlv1.SchemeGroupVersion.Group,
		APIVersion: mlv1.SchemeGroupVersion.Version,
		ObjectType: &mlv1.DatasetVersion{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}
