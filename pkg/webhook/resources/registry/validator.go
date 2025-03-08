package registry

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

type modelValidator struct {
	admission.DefaultValidator

	registryCache ctlmlv1.RegistryCache
}

var _ admission.Validator = &modelValidator{}

func NewModelValidator(mgmt *config.Management) admission.Validator {
	return &modelValidator{
		registryCache: mgmt.LLMFactory.Ml().V1().Registry().Cache(),
	}
}

func (v *modelValidator) Create(_ *admission.Request, obj runtime.Object) error {
	m := obj.(*mlv1.Model)

	// Verify if the registry exists
	if _, err := v.registryCache.Get(m.Spec.Registry); err != nil {
		if errors.IsNotFound(err) {
			return werror.BadRequest(fmt.Sprintf("registry %s not found", m.Spec.Registry))
		}
		return werror.InternalError(fmt.Sprintf("get registry %s failed: %v", m.Spec.Registry, err))
	}

	return nil
}

func (v *modelValidator) Update(_ *admission.Request, oldObj runtime.Object, newObj runtime.Object) error {
	oldM := oldObj.(*mlv1.Model)
	newM := newObj.(*mlv1.Model)

	if oldM.Spec.Registry != newM.Spec.Registry {
		return werror.MethodNotAllowed("registry field cannot be modified once set")
	}

	return nil
}

func (v *modelValidator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"models"},
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   mlv1.SchemeGroupVersion.Group,
		APIVersion: mlv1.SchemeGroupVersion.Version,
		ObjectType: &mlv1.Model{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}

type datasetValidator struct {
	admission.DefaultValidator
	registryCache ctlmlv1.RegistryCache
}

var _ admission.Validator = &datasetValidator{}

func NewDatasetValidator(mgmt *config.Management) admission.Validator {
	return &datasetValidator{
		registryCache: mgmt.LLMFactory.Ml().V1().Registry().Cache(),
	}
}

func (v *datasetValidator) Create(_ *admission.Request, obj runtime.Object) error {
	d := obj.(*mlv1.Dataset)

	if _, err := v.registryCache.Get(d.Spec.Registry); err != nil {
		if errors.IsNotFound(err) {
			return werror.BadRequest(fmt.Sprintf("registry %s not found", d.Spec.Registry))
		}
		return werror.InternalError(fmt.Sprintf("get registry %s failed: %v", d.Spec.Registry, err))
	}

	return nil
}

func (v *datasetValidator) Update(_ *admission.Request, oldObj runtime.Object, newObj runtime.Object) error {
	oldD := oldObj.(*mlv1.Dataset)
	newD := newObj.(*mlv1.Dataset)

	if oldD.Spec.Registry != newD.Spec.Registry {
		return werror.MethodNotAllowed("registry field cannot be modified once set")
	}

	return nil
}

func (v *datasetValidator) Resource() admission.Resource {
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

type modelVersionValidator struct {
	admission.DefaultValidator

	modelCache ctlmlv1.ModelCache
}

var _ admission.Validator = &modelVersionValidator{}

func NewModelVersionValidator(mgmt *config.Management) admission.Validator {
	return &modelVersionValidator{
		modelCache: mgmt.LLMFactory.Ml().V1().Model().Cache(),
	}
}

func (v *modelVersionValidator) Create(_ *admission.Request, obj runtime.Object) error {
	mv := obj.(*mlv1.ModelVersion)

	if _, err := v.modelCache.Get(mv.Namespace, mv.Spec.Model); err != nil {
		if errors.IsNotFound(err) {
			return werror.BadRequest(fmt.Sprintf("model %s/%s not found", mv.Namespace, mv.Spec.Model))
		}
		return werror.InternalError(fmt.Sprintf("get model %s/%s failed: %v", mv.Namespace, mv.Name, err))
	}

	return nil
}

func (v *modelVersionValidator) Update(_ *admission.Request, oldObj runtime.Object, newObj runtime.Object) error {
	oldMV := oldObj.(*mlv1.ModelVersion)
	newMV := newObj.(*mlv1.ModelVersion)

	if oldMV.Spec.Model != newMV.Spec.Model || oldMV.Spec.Version != newMV.Spec.Version {
		return werror.MethodNotAllowed("model and version field cannot be modified once set")
	}

	if !reflect.DeepEqual(oldMV.Spec.CopyFrom, newMV.Spec.CopyFrom) {
		return werror.MethodNotAllowed("copyFrom field cannot be modified once set")
	}

	return nil
}

func (v *modelVersionValidator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"modelversions"},
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   mlv1.SchemeGroupVersion.Group,
		APIVersion: mlv1.SchemeGroupVersion.Version,
		ObjectType: &mlv1.ModelVersion{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}

type datasetVersionValidator struct {
	admission.DefaultValidator

	datasetCache ctlmlv1.DatasetCache
}

var _ admission.Validator = &datasetVersionValidator{}

func NewDatasetVersionValidator(mgmt *config.Management) admission.Validator {
	return &datasetVersionValidator{
		datasetCache: mgmt.LLMFactory.Ml().V1().Dataset().Cache(),
	}
}

func (v *datasetVersionValidator) Create(_ *admission.Request, obj runtime.Object) error {
	dv := obj.(*mlv1.DatasetVersion)

	if _, err := v.datasetCache.Get(dv.Namespace, dv.Spec.Dataset); err != nil {
		if errors.IsNotFound(err) {
			return werror.BadRequest(fmt.Sprintf("dataset %s/%s not found", dv.Namespace, dv.Spec.Dataset))
		}
		return werror.InternalError(fmt.Sprintf("get dataset %s/%s failed: %v", dv.Namespace, dv.Name, err))
	}

	return nil
}

func (v *datasetVersionValidator) Update(_ *admission.Request, oldObj runtime.Object, newObj runtime.Object) error {
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

func (v *datasetVersionValidator) Resource() admission.Resource {
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
