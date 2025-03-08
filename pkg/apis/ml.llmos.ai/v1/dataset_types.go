package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Registry",type="string",JSONPath=`.spec.registry`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Dataset is a definition for the LLM Dataset
type Dataset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatasetSpec   `json:"spec,omitempty"`
	Status DatasetStatus `json:"status,omitempty"`
}

// DatasetSpec defines the desired state of Dataset
type DatasetSpec struct {
	// +optional
	Card DatasetCard `json:"datasetCard"`

	Registry string           `json:"registry"`
	Versions []DatasetVersion `json:"versions,omitempty"`
}

// DatasetVersion represents a specific version of a dataset
type DatasetVersion struct {
	Version           string `json:"version"`
	EnableFastLoading bool   `json:"enableFastLoading"`
}

// DatasetCard contains metadata and description for a dataset
// Reference: https://huggingface.co/docs/datasets/dataset_card
type DatasetCard struct {
	Description string          `json:"description"`
	MetaData    DatasetMetaData `json:"metadata"`
}

// DatasetMetaData is the metadata of a dataset
type DatasetMetaData struct {
	Tags        []string `json:"tags,omitempty"`        // Tags associated with the dataset
	License     string   `json:"license,omitempty"`     // License under which the dataset is released
	SplitTypes  []string `json:"splitTypes,omitempty"`  // Types of splits (e.g., train, test, validation)
	Features    []string `json:"features,omitempty"`    // Features included in the dataset
	NumSamples  int      `json:"numSamples,omitempty"`  // Total number of samples in the dataset
	Language    string   `json:"language,omitempty"`    // Language of the dataset
	Citation    string   `json:"citation,omitempty"`    // Citation information for the dataset
	Homepage    string   `json:"homepage,omitempty"`    // Homepage of the dataset
	DownloadURL string   `json:"downloadUrl,omitempty"` // URL to download the dataset
	Authors     []string `json:"authors,omitempty"`     // Authors of the dataset
	Contact     string   `json:"contact,omitempty"`     // Contact information for the dataset
}

// DatasetStatus defines the observed state of Dataset
type DatasetStatus struct {
	Conditions    []common.Condition     `json:"conditions,omitempty"`    // Conditions of the dataset
	VersionStatus []DatasetVersionStatus `json:"versionStatus,omitempty"` // Status of each dataset version
}

// DatasetVersionStatus defines the observed state of DatasetVersion
type DatasetVersionStatus struct {
	Version string `json:"version"` // Version of the dataset
	Address string `json:"address"` // Address where the dataset is hosted
	// +optional
	Snapshot string `json:"snapshot,omitempty"` // Snapshot of the dataset version
}
