package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="MIN UPGRADE VERSION",type="string",JSONPath=`.spec.minUpgradableVersion`
// +kubebuilder:printcolumn:name="RELEASE DATE",type="string",JSONPath=`.spec.releaseDate`
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Version is the Schema for the upgrade version
type Version struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VersionSpec `json:"spec,omitempty"`
}

// VersionSpec defines the desired state of Version
type VersionSpec struct {
	// +optional, Specify the minimum version that can be upgraded from
	MinUpgradableVersion string `json:"minUpgradableVersion,omitempty"`

	// +optional, Specify the kubernetes version to upgrade to
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`

	// +kubebuilder:validation:Required
	ReleaseDate string `json:"releaseDate,omitempty"`

	// +optional, Specify the tags of the version
	Tags []string `json:"tags,omitempty"`
}
