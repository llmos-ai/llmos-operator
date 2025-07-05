package snapshotting

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
)

const (
	ResourceTypeLabel = "llmos.ai/resource-type"
)

type ResourceHandler interface {
	GetSnapshottingStatus(namespace, name string) (*mlv1.SnapshottingStatus, error)
	UpdateSnapshottingStatus(namespace, name string, status *mlv1.SnapshottingStatus) error
	GetContentSize(ctx context.Context, namespace, name string) (int64, error)
	GetLatestReadySnapshot(namespace, localModelName string) (string, error)
	// GetResourceType returns the resource type for event handler naming
	GetResourceType() string
}

// Spec contains the key spec to create a snapshot including pvc, job, volumesnapshot
type Spec struct {
	// Namespace is the namespace where resources will be created
	Namespace string `json:"namespace"`
	// Name is the base name for created resources
	Name string `json:"name"`
	// Labels to be applied to all created resources
	Labels map[string]string `json:"labels,omitempty"`
	// OwnerReferences to be applied to all created resources
	OwnerReferences []metav1.OwnerReference `json:"ownerReferences,omitempty"`

	// PVC configuration
	PVCSpec PVCSpec `json:"pvcSpec"`
	// Job configuration
	JobSpec JobSpec `json:"jobSpec"`
}

type PVCSpec struct {
	// Size of the PVC
	Size string `json:"size"`
	// AccessModes for the PVC
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`
	// RestoreFromLatestSnapshot indicates whether to restore from the latest snapshot
	RestoreFromLatestSnapshot bool `json:"restoreFromLatestSnapshot,omitempty"`
}

// JobSpec defines the specification for creating a Job
type JobSpec struct {
	BackoffLimit            *int32
	TTLSecondsAfterFinished *int32
	Image                   string
	Args                    []string
}
