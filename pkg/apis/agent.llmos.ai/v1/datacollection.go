package v1

import (
	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=dc;dcs
// +kubebuilder:printcolumn:name="Registry",type="string",JSONPath=`.spec.registry`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// DataCollection is a definition for the application data
type DataCollection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataCollectionSpec   `json:"spec,omitempty"`
	Status DataCollectionStatus `json:"status,omitempty"`
}

type DataCollectionSpec struct {
	Registry string `json:"registry"`
}

type DataCollectionStatus struct {
	RootPath   string             `json:"rootPath"`
	Files      []FileInfo         `json:"files,omitempty"`
	Conditions []common.Condition `json:"conditions,omitempty"`
}
