package v1

import (
	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Registry",type="string",JSONPath=`.spec.registry`
// +kubebuilder:printcolumn:name="ModelName",type="string",JSONPath=`.spec.modelName`
// +kubebuilder:printcolumn:name="DefaultVersion",type="integer",JSONPath=`.status.defaultVersion`
// +kubebuilder:printcolumn:name="DefaultVersionName",type="string",JSONPath=`.status.defaultVersionName`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// LocalModel is the model stored in the local storage
// The LocalModel acts as a parent resource for LocalModelVersion instances and provides
// registry/source information for downloading the model data.
type LocalModel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              LocalModelSpec   `json:"spec,omitempty"`
	Status            LocalModelStatus `json:"status,omitempty"`
}

type LocalModelSpec struct {
	// Registry can be the private registry or the public registry like huggingface.co
	Registry string `json:"registry"`
	// ModelName is the name of the model in the registry like deepseek-ai/deepseek-r1
	ModelName string `json:"modelName"`
	// +optional
	// DefaultVersion is the default version of the local model
	// If DefaultVersion is empty, choose the latest version
	DefaultVersion string `json:"defaultVersion"`
}

type LocalModelStatus struct {
	// Conditions is a list of conditions representing the status of the Model
	Conditions         []common.Condition `json:"conditions,omitempty"`
	DefaultVersion     int                `json:"defaultVersion,omitempty"`
	DefaultVersionName string             `json:"defaultVersionName,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="LocalModel",type="string",JSONPath=`.spec.localModel`
// +kubebuilder:printcolumn:name="Version",type=integer,priority=8,JSONPath=`.status.version`
// +kubebuilder:printcolumn:name="VolumeSnapshot",type="string",JSONPath=`.status.volumeSnapshot`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// LocalModelVersion is the version of local model
// It references its parent LocalModel via the localModel field.
// The controller will create appropriate PVCs and download jobs to fetch the model contents.
type LocalModelVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              LocalModelVersionSpec   `json:"spec,omitempty"`
	Status            LocalModelVersionStatus `json:"status,omitempty"`
}

type LocalModelVersionSpec struct {
	LocalModel string `json:"localModel"`
}

type LocalModelVersionStatus struct {
	Version int `json:"version"`
	// Conditions is a list of conditions representing the status of the Model
	Conditions []common.Condition `json:"conditions,omitempty"`
	// +optional
	VolumeSnapshot string `json:"volumeSnapshot,omitempty"`
	// +optional
	SnapshottingStatus SnapshottingStatus `json:"snapshottingStatus,omitempty"`
}
