package datasetversion

import (
	"context"
	"fmt"
	"reflect"

	"github.com/oneblock-ai/webhook/pkg/server/admission"
	corev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	ctlstoragev1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/storage.k8s.io/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/config"
	werror "github.com/llmos-ai/llmos-operator/pkg/webhook/error"
)

type validator struct {
	admission.DefaultValidator

	ctx context.Context

	datasetCache        ctlmlv1.DatasetCache
	datasetVersionCache ctlmlv1.DatasetVersionCache
	registryCache       ctlmlv1.RegistryCache
	secretCache         corev1.SecretCache
	storageClassCache   ctlstoragev1.StorageClassCache
	rm                  *registry.Manager
}

var _ admission.Validator = &validator{}

func NewValidator(mgmt *config.Management) admission.Validator {
	v := &validator{
		ctx:                 context.Background(),
		datasetCache:        mgmt.LLMFactory.Ml().V1().Dataset().Cache(),
		datasetVersionCache: mgmt.LLMFactory.Ml().V1().DatasetVersion().Cache(),
		registryCache:       mgmt.LLMFactory.Ml().V1().Registry().Cache(),
		secretCache:         mgmt.CoreFactory.Core().V1().Secret().Cache(),
		storageClassCache:   mgmt.StorageFactory.Storage().V1().StorageClass().Cache(),
	}
	v.rm = registry.NewManager(v.secretCache.Get, v.registryCache.Get)
	return v
}

func (v *validator) Create(_ *admission.Request, obj runtime.Object) error {
	dv := obj.(*mlv1.DatasetVersion)

	if dv.Spec.Publish {
		if dv.Spec.CopyFrom == nil || dv.Spec.CopyFrom.Version == "" ||
			dv.Spec.CopyFrom.Dataset == "" || dv.Spec.CopyFrom.Namespace == "" {
			return werror.BadRequest("copyFrom field is required when publish is true")
		}
	}

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

	// If the dataset version has no content, it's not allowed to be published.
	if !oldDV.Spec.Publish && newDV.Spec.Publish {
		// Check if the dataset version has content before allowing publish
		if err := v.checkDatasetVersionHasContent(newDV); err != nil {
			return werror.BadRequest(fmt.Sprintf("cannot publish dataset version: %v", err))
		}
	}

	if oldDV.Spec.Dataset != newDV.Spec.Dataset || oldDV.Spec.Version != newDV.Spec.Version {
		return werror.MethodNotAllowed("dataset and version field cannot be modified once set")
	}

	if !reflect.DeepEqual(oldDV.Spec.CopyFrom, newDV.Spec.CopyFrom) {
		return werror.MethodNotAllowed("copyFrom field cannot be modified once set")
	}

	return nil
}

// checkDatasetVersionHasContent checks if the dataset version has any content
func (v *validator) checkDatasetVersionHasContent(dv *mlv1.DatasetVersion) error {
	// Check if the dataset version is ready and has a root path
	if !mlv1.Ready.IsTrue(dv) {
		return fmt.Errorf("dataset version is not ready")
	}

	if dv.Status.RootPath == "" {
		return fmt.Errorf("dataset version root path is empty")
	}

	if dv.Status.Registry == "" {
		return fmt.Errorf("dataset version registry is empty")
	}

	// Create backend client to check content size
	b, err := v.rm.NewBackendFromRegistry(v.ctx, dv.Status.Registry)
	if err != nil {
		return fmt.Errorf("failed to create backend client: %w", err)
	}

	// Get the total size of content in the dataset version directory
	totalSize, err := b.GetSize(v.ctx, dv.Status.RootPath)
	if err != nil {
		return fmt.Errorf("failed to get size: %w", err)
	}

	// If total size is 0, it means there's no content
	if totalSize == 0 {
		return fmt.Errorf("dataset version has no content")
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
