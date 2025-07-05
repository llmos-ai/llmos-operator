package v1

import (
	"github.com/llmos-ai/llmos-operator/pkg/utils/condition"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Volume struct {
	Name string                           `json:"name"`
	Spec corev1.PersistentVolumeClaimSpec `json:"spec"`
}

var (
	Ready condition.Cond = "ready"
)

type SnapshottingPhase string

const (
	SnapshottingPhasePreparePVC    SnapshottingPhase = "PreparePVC"
	SnapshottingPhasePVCReady      SnapshottingPhase = "PVCReady"
	SnapshottingPhaseDownloading   SnapshottingPhase = "Downloading"
	SnapshottingPhaseDownloaded    SnapshottingPhase = "Downloaded"
	SnapshottingPhaseSnapshotting  SnapshottingPhase = "Snapshotting"
	SnapshottingPhaseSnapshotReady SnapshottingPhase = "SnapshotReady"
	SnapshottingPhaseFailed        SnapshottingPhase = "Failed"
)

type SnapshottingStatus struct {
	// +optional
	// Phase is the phase of the VolumeSnapshot
	Phase SnapshottingPhase `json:"phase,omitempty"`
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// +optional
	Message string `json:"message,omitempty"`
	// +optional
	// SnapshotName is the name of the VolumeSnapshot created for this model version
	SnapshotName string `json:"snapshotName,omitempty"`
	// +optional
	// PVCName is the name of the PVC created for this model version
	PVCName string `json:"pvcName,omitempty"`
	// +optional
	// JobName is the name of the Job created for downloading the model
	JobName string `json:"jobName,omitempty"`
}
