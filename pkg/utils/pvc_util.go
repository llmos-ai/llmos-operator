package utils

import (
	"encoding/json"
	"reflect"

	ctlcorev1 "github.com/rancher/wrangler/v2/pkg/generated/controllers/core/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
)

type PVCHandler struct {
	pvcs     ctlcorev1.PersistentVolumeClaimClient
	pvcCache ctlcorev1.PersistentVolumeClaimCache
}

func NewPVCHandler(pvcs ctlcorev1.PersistentVolumeClaimClient,
	pvcCache ctlcorev1.PersistentVolumeClaimCache) *PVCHandler {
	return &PVCHandler{
		pvcs:     pvcs,
		pvcCache: pvcCache,
	}
}

// CreatePVCFromAnnotation helps to create PVCs from the annotation
func (h *PVCHandler) CreatePVCFromAnnotation(pvcTemplates, namespace string, ownerRefs []metav1.OwnerReference) error {

	var pvcs []*corev1.PersistentVolumeClaim
	if err := json.Unmarshal([]byte(pvcTemplates), &pvcs); err != nil {
		return err
	}

	var (
		pvc *corev1.PersistentVolumeClaim
		err error
	)

	for _, annoPVC := range pvcs {
		if pvc, err = h.pvcCache.Get(namespace, annoPVC.Name); err != nil {
			if apierrors.IsNotFound(err) {
				annoPVC.Namespace = namespace
				annoPVC.OwnerReferences = ownerRefs
				if _, err = h.pvcs.Create(annoPVC); err != nil {
					return err
				}
				continue
			}
			return err
		}

		// users may also resize the volumes outside the annotation. In that case, we can't track the update.
		// If the storage request in the cluster annotation is less or equal to the actual Volume size, just skip it.
		if annoPVC.Spec.Resources.Requests.Storage().Cmp(*pvc.Spec.Resources.Requests.Storage()) <= 0 {
			continue
		}

		toUpdate := pvc.DeepCopy()
		toUpdate.Spec.Resources.Requests = annoPVC.Spec.Resources.Requests
		if !reflect.DeepEqual(toUpdate, pvc) {
			if _, err = h.pvcs.Update(toUpdate); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *PVCHandler) CreatePVCByVolume(pvcs []mlv1.Volume, namespace string, ownerRefs []metav1.OwnerReference) error {
	var (
		pvc *corev1.PersistentVolumeClaim
		err error
	)

	for _, volume := range pvcs {
		if pvc, err = h.pvcCache.Get(namespace, volume.Name); err != nil {
			if apierrors.IsNotFound(err) {
				pvc = &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      volume.Name,
						Namespace: namespace,
					},
					Spec: volume.Spec,
				}
				if ownerRefs != nil {
					pvc.OwnerReferences = ownerRefs
				}
				if _, err = h.pvcs.Create(pvc); err != nil {
					return err
				}
				continue
			}
			return err
		}

		// users may also resize the volumes outside the annotation. In that case, we can't track the update.
		// If the storage request in the cluster annotation is less or equal to the actual Volume size, just skip it.
		if volume.Spec.Resources.Requests.Storage().Cmp(*pvc.Spec.Resources.Requests.Storage()) <= 0 {
			continue
		}

		toUpdate := pvc.DeepCopy()
		toUpdate.Spec.Resources.Requests = volume.Spec.Resources.Requests
		if !reflect.DeepEqual(toUpdate, pvc) {
			if _, err = h.pvcs.Update(toUpdate); err != nil {
				return err
			}
		}
	}
	return nil
}
