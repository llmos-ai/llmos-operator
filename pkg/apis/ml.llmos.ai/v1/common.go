package v1

import (
	corev1 "k8s.io/api/core/v1"
)

type Volume struct {
	Name string                           `json:"name"`
	Spec corev1.PersistentVolumeClaimSpec `json:"spec"`
}

type CopyFrom struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

type Version struct {
	Version    string `json:"version"`
	ObjectName string `json:"objectName"`
}
