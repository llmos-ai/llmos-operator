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

// Model is a definition for the LLM Model
type Model struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModelSpec   `json:"spec,omitempty"`
	Status ModelStatus `json:"status,omitempty"`
}

// ModelSpec defines the desired state of Model
type ModelSpec struct {
	// +optional
	Card ModelCard `json:"modelCard"`

	Registry string         `json:"registry"`
	Versions []ModelVersion `json:"versions,omitempty"`
}

// ModelVersion represents a specific version of a model
type ModelVersion struct {
	Version           string `json:"version"`
	EnableFastLoading bool   `json:"enableFastLoading"`
}

// ModelCard contains metadata and description for a model
// Reference: https://huggingface.co/docs/hub/models-cards
type ModelCard struct {
	Description string        `json:"description"`
	MetaData    ModelMetaData `json:"metadata"`
}

// ModelMetaData is the metadata of a model
type ModelMetaData struct {
	Tags              []string `json:"tags,omitempty"`               // Tags associated with the model
	License           string   `json:"license,omitempty"`            // License under which the model is released
	Datasets          []string `json:"datasets,omitempty"`           // Datasets used for training the model
	Metrics           []string `json:"metrics,omitempty"`            // Metrics used for evaluation
	Language          string   `json:"language,omitempty"`           // Programming language used in the model
	LibraryName       string   `json:"libraryName,omitempty"`        // Name of the library used
	LibraryVersion    string   `json:"libraryVersion,omitempty"`     // Version of the library used
	CPU               bool     `json:"cpu,omitempty"`                // Whether the model supports CPU
	GPU               bool     `json:"gpu,omitempty"`                // Whether the model supports GPU
	Framework         string   `json:"framework,omitempty"`          // Framework used for the model
	TrainingData      string   `json:"trainingData,omitempty"`       // Description of the training data
	EvaluationResults string   `json:"evaluationResults,omitempty"`  // Results of the model evaluation
	BaseModel         string   `json:"baseModel,omitempty"`          // Base model information
}

// ModelStatus defines the observed state of Model
type ModelStatus struct {
	Conditions    []common.Condition   `json:"conditions,omitempty"`    // Conditions of the model
	VersionStatus []ModelVersionStatus `json:"versionStatus,omitempty"` // Status of each model version
}

// ModelVersionStatus defines the observed state of ModelVersion
type ModelVersionStatus struct {
	Version string `json:"version"` // Version of the model
	Address string `json:"address"` // Address where the model is hosted
	// +optional
	Snapshot string `json:"snapshot,omitempty"` // Snapshot of the model version
}
