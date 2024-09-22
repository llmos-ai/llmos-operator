package raycluster

import (
	"encoding/json"
	"reflect"

	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
)

// createPVCFromAnnotation helps to create PVCs from the cluster annotation
func (h *handler) createPVCFromAnnotation(_ string, cluster *rayv1.RayCluster) (*rayv1.RayCluster, error) {
	if cluster == nil || cluster.DeletionTimestamp != nil {
		return nil, nil
	}

	pvcTemplates, ok := cluster.Annotations[constant.AnnotationVolumeClaimTemplates]
	if !ok || pvcTemplates == "" {
		return nil, nil
	}

	var pvcs []*corev1.PersistentVolumeClaim
	if err := json.Unmarshal([]byte(pvcTemplates), &pvcs); err != nil {
		return nil, err
	}

	var (
		pvc *corev1.PersistentVolumeClaim
		err error
	)

	for _, annoPVC := range pvcs {
		if pvc, err = h.pvcCache.Get(cluster.Namespace, annoPVC.Name); err != nil {
			if apierrors.IsNotFound(err) {
				annoPVC.Namespace = cluster.Namespace
				annoPVC.OwnerReferences = []metav1.OwnerReference{
					{
						APIVersion: cluster.APIVersion,
						Kind:       cluster.Kind,
						Name:       cluster.Name,
						UID:        cluster.UID,
					},
				}
				if _, err = h.pvcs.Create(annoPVC); err != nil {
					return nil, err
				}
				continue
			}
			return nil, err
		}

		// users may also resize the volumes outside the annotation. In that case, we can't track the update.
		// If the storage request in the annotation is less or equal to the actual PVC size, just skip it.
		if annoPVC.Spec.Resources.Requests.Storage().Cmp(*pvc.Spec.Resources.Requests.Storage()) <= 0 {
			continue
		}

		toUpdate := pvc.DeepCopy()
		toUpdate.Spec.Resources.Requests = annoPVC.Spec.Resources.Requests
		if !reflect.DeepEqual(toUpdate, pvc) {
			if _, err = h.pvcs.Update(toUpdate); err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}
