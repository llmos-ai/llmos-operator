package notebook

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
)

// createNoteBookPVC helps to create PVCs from the resource annotation
func (h *Handler) createNoteBookPVC(_ string, notebook *mlv1.Notebook) (*mlv1.Notebook, error) {
	if notebook == nil || notebook.DeletionTimestamp != nil {
		return nil, nil
	}

	if notebook.Spec.Volumes == nil || len(notebook.Spec.Volumes) == 0 {
		return nil, nil
	}

	ownerReferences := []metav1.OwnerReference{
		{
			APIVersion: notebook.APIVersion,
			Kind:       notebook.Kind,
			Name:       notebook.Name,
			UID:        notebook.UID,
		},
	}

	err := h.pvcHandler.CreatePVCByVolume(notebook.Spec.Volumes, notebook.Namespace, ownerReferences)
	if err != nil {
		return notebook, err
	}
	return notebook, nil
}
