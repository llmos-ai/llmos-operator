package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Volume struct {
	Name string                           `json:"name"`
	Spec corev1.PersistentVolumeClaimSpec `json:"spec"`
}

// CacheStatusType defines the status of the local cache operation
// +kubebuilder:validation:Enum=idle;pending;downloading;completed;failed
type CacheStatusType string

const (
	// CacheStatusIdle indicates no cache operation is in progress
	CacheStatusIdle CacheStatusType = "idle"
	// CacheStatusPending indicates a cache operation has been requested but not started
	CacheStatusPending CacheStatusType = "pending"
	// CacheStatusDownloading indicates a cache operation is in progress
	CacheStatusDownloading CacheStatusType = "downloading"
	// CacheStatusCompleted indicates a cache operation has completed successfully
	CacheStatusCompleted CacheStatusType = "completed"
	// CacheStatusFailed indicates a cache operation has failed
	CacheStatusFailed CacheStatusType = "failed"
)

type CacheStatus struct {
	// Status represents the current status of the local cache operation
	// +optional
	Status CacheStatusType `json:"status"`
	// JobName is the name of the job responsible for the cache operation
	// +optional
	JobName string `json:"jobName,omitempty"`
	// VolumeSnapshot stores the cached content and is used to restore the model or dataset
	// +optional
	VolumeSnapshot string `json:"volumeSnapshot,omitempty"`
	// LastCacheTime is the timestamp when the model was last successfully cached
	// +optional
	LastCacheTime *metav1.Time `json:"lastCacheTime,omitempty"`
	// CacheMessage provides additional information about the cache operation
	// +optional
	CacheMessage string `json:"cacheMessage,omitempty"`
}

// LocalCache defines the state of the cache, using the same enumeration values ​​as CacheStateType
// +kubebuilder:validation:Enum=active;inactive
type CacheStateType string

const (
	// CacheStateActive represents the active state of the cache
	CacheStateActive CacheStateType = "active"
	// CacheStateInactive represents the inactive state of the cache
	CacheStateInactive CacheStateType = "inactive"
)
