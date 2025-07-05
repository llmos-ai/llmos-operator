package notebook

import (
	"fmt"

	"github.com/oneblock-ai/webhook/pkg/server/admission"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/config"
)

type validator struct {
	admission.DefaultValidator

	datasetVersionCache ctlmlv1.DatasetVersionCache
}

var _ admission.Validator = &validator{}

func NewValidator(mgmt *config.Management) admission.Validator {
	return &validator{
		datasetVersionCache: mgmt.LLMFactory.Ml().V1().DatasetVersion().Cache(),
	}
}

func (v *validator) Create(_ *admission.Request, newObj runtime.Object) error {
	notebook := newObj.(*mlv1.Notebook)

	if err := v.validateDatasetMountings(notebook); err != nil {
		return err
	}

	return validateVolumeClaimTemplatesAnnotation(notebook)
}

func (v *validator) Update(_ *admission.Request, _, newObj runtime.Object) error {
	notebook := newObj.(*mlv1.Notebook)

	if err := v.validateDatasetMountings(notebook); err != nil {
		return err
	}

	return validateVolumeClaimTemplatesAnnotation(notebook)
}

func validateVolumeClaimTemplatesAnnotation(cluster *mlv1.Notebook) error {
	volumeClaimTemplates, ok := cluster.Annotations[constant.AnnotationVolumeClaimTemplates]
	if !ok || volumeClaimTemplates == "" {
		return nil
	}
	return utils.ValidateVolumeClaimTemplatesAnnotation(volumeClaimTemplates)
}

func (v *validator) validateDatasetMountings(notebook *mlv1.Notebook) error {
	for _, mounting := range notebook.Spec.DatasetMountings {
		// Find the DatasetVersion by iterating through all DatasetVersions in the namespace
		datasetVersions, err := v.datasetVersionCache.List(notebook.Namespace, labels.SelectorFromSet(map[string]string{
			constant.LabelDatasetName:    mounting.DatasetName,
			constant.LabelDatasetVersion: mounting.Version,
		}))
		if err != nil {
			return fmt.Errorf("failed to list dataset versions: %w", err)
		}
		if len(datasetVersions) != 1 {
			return fmt.Errorf("found %d dataset version from %s/%s",
				len(datasetVersions), mounting.DatasetName, mounting.Version)
		}
		if !datasetVersions[0].Spec.Publish ||
			datasetVersions[0].Status.PublishStatus.Phase != mlv1.SnapshottingPhaseSnapshotReady {
			return fmt.Errorf("dataset version %s/%s has not been published", mounting.DatasetName, mounting.Version)
		}
	}

	return nil
}

func (v *validator) Resource() admission.Resource {
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
