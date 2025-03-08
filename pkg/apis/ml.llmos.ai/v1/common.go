package v1

import (
	corev1 "k8s.io/api/core/v1"
)

type Volume struct {
	Name string                           `json:"name"`
	Spec corev1.PersistentVolumeClaimSpec `json:"spec"`
}
