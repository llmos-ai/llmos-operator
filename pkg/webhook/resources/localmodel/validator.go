package localmodel

import (
	"fmt"

	"github.com/oneblock-ai/webhook/pkg/server/admission"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/config"
)

type validator struct {
	admission.DefaultValidator

	localModelVersionCache ctlmlv1.LocalModelVersionCache
}

var _ admission.Validator = &validator{}

func NewValidator(mgmt *config.Management) admission.Validator {
	return &validator{
		localModelVersionCache: mgmt.LLMFactory.Ml().V1().LocalModelVersion().Cache(),
	}
}

func (v *validator) Create(_ *admission.Request, obj runtime.Object) error {
	lm := obj.(*mlv1.LocalModel)

	ready, err := v.isLocalModelVersionReady(lm.Namespace, lm.Spec.DefaultVersion)
	if err != nil {
		return err
	}
	if !ready {
		return fmt.Errorf("local model version %s is not ready", lm.Spec.DefaultVersion)
	}

	return nil
}

func (v *validator) Update(_ *admission.Request, oldObj runtime.Object, newObj runtime.Object) error {
	lm := newObj.(*mlv1.LocalModel)

	ready, err := v.isLocalModelVersionReady(lm.Namespace, lm.Spec.DefaultVersion)
	if err != nil {
		return err
	}
	if !ready {
		return fmt.Errorf("local model version %s is not ready", lm.Spec.DefaultVersion)
	}

	return nil
}

func (v *validator) isLocalModelVersionReady(namespace, versionName string) (bool, error) {
	if versionName == "" {
		return true, nil
	}

	lmv, err := v.localModelVersionCache.Get(namespace, versionName)
	if err != nil {
		return false, fmt.Errorf("failed to get local model version %s/%s: %v", namespace, versionName, err)
	}
	if !mlv1.Ready.IsTrue(lmv) {
		return false, nil
	}

	return true, nil
}

func (v *validator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"localmodels"},
		Scope:      admissionregv1.NamespacedScope,
		APIGroup:   mlv1.SchemeGroupVersion.Group,
		APIVersion: mlv1.SchemeGroupVersion.Version,
		ObjectType: &mlv1.LocalModel{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}
