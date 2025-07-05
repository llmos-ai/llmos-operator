package v1

import (
	"github.com/llmos-ai/llmos-operator/pkg/apis/common"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	Card *DatasetCard `json:"datasetCard,omitempty"`

	Registry string `json:"registry"`
}

type DatasetStatus struct {
	// Conditions is a list of conditions representing the status of the Dataset
	Conditions []common.Condition `json:"conditions,omitempty"`
	// RootPath is the root path of the dataset in the storage
	RootPath string `json:"path,omitempty"`
	// Versions is a list of versions of the dataset
	Versions []Version `json:"versions,omitempty"`
}

// DatasetCard contains metadata and description for a dataset
// Reference: https://huggingface.co/docs/datasets/dataset_card
type DatasetCard struct {
	// +optional
	Description string          `json:"description,omitempty"`
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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=dv;dvs
// +kubebuilder:printcolumn:name="Dataset",type="string",JSONPath=`.spec.dataset`
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="VolumeSnapshot",type="string",JSONPath=`.status.publishStatus.snapshotName`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// DatasetVersion is a definition for the LLM Dataset Version
type DatasetVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatasetVersionSpec   `json:"spec,omitempty"`
	Status DatasetVersionStatus `json:"status,omitempty"`
}

// DatasetVersionSpec defines the desired state of DatasetVersion
type DatasetVersionSpec struct {
	Dataset string `json:"dataset"`
	Version string `json:"version"`
	// +optional
	CopyFrom *CopyFrom `json:"copyFrom,omitempty"`
	// +optional
	Publish bool `json:"publish"`
}

type DatasetVersionStatus struct {
	Conditions []common.Condition `json:"conditions,omitempty"`

	Registry string `json:"registry"`
	RootPath string `json:"rootPath"`
	// +optional
	PublishStatus SnapshottingStatus `json:"publishStatus"`
}

type CopyFrom struct {
	Namespace string `json:"namespace"`
	Dataset   string `json:"dataset"`
	Version   string `json:"version"`
}

type Version struct {
	Version    string `json:"version"`
	ObjectName string `json:"objectName"`
}
