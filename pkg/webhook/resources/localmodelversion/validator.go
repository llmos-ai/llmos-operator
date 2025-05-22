package localmodelversion

import (
	"fmt"
	"strings"

	ctlstoragev1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/storage.k8s.io/v1"
	"github.com/oneblock-ai/webhook/pkg/server/admission"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/config"
)

const (
	storageClassName = "llmos-ceph-block"

	huggingfaceRegistry = "huggingface"
	modelScopeRegistry  = "modelscope"
)

type validator struct {
	admission.DefaultValidator

	localModelCache   ctlmlv1.LocalModelCache
	modelCache        ctlmlv1.ModelCache
	storageClassCache ctlstoragev1.StorageClassCache
}

var _ admission.Validator = &validator{}

func NewValidator(mgmt *config.Management) admission.Validator {
	return &validator{
		localModelCache:   mgmt.LLMFactory.Ml().V1().LocalModel().Cache(),
		modelCache:        mgmt.LLMFactory.Ml().V1().Model().Cache(),
		storageClassCache: mgmt.StorageFactory.Storage().V1().StorageClass().Cache(),
	}
}

func (v *validator) Create(_ *admission.Request, obj runtime.Object) error {
	lmv := obj.(*mlv1.LocalModelVersion)

	if err := v.checkStorageClassExists(); err != nil {
		return err
	}

	// Validate that the referenced LocalModel exists
	localModelName := lmv.Spec.LocalModel
	localModel, err := v.localModelCache.Get(lmv.Namespace, localModelName)
	if err != nil {
		return fmt.Errorf("referenced LocalModel '%s' does not exist in namespace '%s': %w",
			localModelName, lmv.Namespace, err)
	}

	return v.isModelReady(localModel.Spec.Registry, localModel.Spec.ModelName)
}

func (v *validator) Update(_ *admission.Request, oldObj runtime.Object, newObj runtime.Object) error {
	oldLMV, ok := oldObj.(*mlv1.LocalModelVersion)
	if !ok {
		return nil
	}

	newLMV, ok := newObj.(*mlv1.LocalModelVersion)
	if !ok {
		return nil
	}

	// It's not allowed to update spec.localModel
	if oldLMV.Spec.LocalModel != newLMV.Spec.LocalModel {
		return fmt.Errorf("updating spec.localModel is not allowed")
	}

	return nil
}

func (v *validator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"localmodelversions"},
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   mlv1.SchemeGroupVersion.Group,
		APIVersion: mlv1.SchemeGroupVersion.Version,
		ObjectType: &mlv1.LocalModelVersion{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}

func (v *validator) checkStorageClassExists() error {
	if _, err := v.storageClassCache.Get(storageClassName); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("storage class %s not found, please enable system storage firstly", storageClassName)
		}
		return fmt.Errorf("failed to get storage class %s: %w", storageClassName, err)
	}
	return nil
}

func (v *validator) isModelReady(registry, modelName string) error {
	if registry == huggingfaceRegistry || registry == modelScopeRegistry {
		return nil
	}

	tmp := strings.Split(modelName, "/")
	if len(tmp) != 2 {
		return fmt.Errorf("invalid model name %s", modelName)
	}
	namespace, name := tmp[0], tmp[1]

	model, err := v.modelCache.Get(namespace, name)
	if err != nil {
		return fmt.Errorf("failed to get model %s: %w", modelName, err)
	}
	if !mlv1.Ready.IsTrue(model) {
		return fmt.Errorf("model %s is not ready yet", modelName)
	}

	return nil
}
