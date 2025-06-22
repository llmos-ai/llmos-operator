package v1

import (
	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
	"github.com/llmos-ai/llmos-operator/pkg/utils/condition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=kb;kbs
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// KnowledgeBase is a definition for the LLM KnowledgeBase
type KnowledgeBase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KnowledgeBaseSpec   `json:"spec,omitempty"`
	Status KnowledgeBaseStatus `json:"status,omitempty"`
}

type KnowledgeBaseSpec struct {
	// EmbeddingModel is from model service including namespace, such as defalt/text-embedding-3-small
	EmbeddingModel string `json:"embeddingModel"`
	// +optional
	ChunkingConfig ChunkingConfig `json:"chunkingConfig,omitempty"`
	// +optional
	ImportingFiles []ImportingFile `json:"importingFiles,omitempty"`
}

type ChunkingConfig struct {
	// +optional
	Size int `json:"size,omitempty"`
	// +optional
	Overlap int `json:"overlap,omitempty"`
}

type ImportingFile struct {
	DataCollectionName string `json:"dataCollectionName"`
	UID                string `json:"uid"`
}

type KnowledgeBaseStatus struct {
	Conditions    []common.Condition `json:"conditions,omitempty"`
	ClassName     string             `json:"className,omitempty"`
	ImportedFiles []ImportedFile     `json:"importedFiles,omitempty"`
}

type ImportedFile struct {
	UID                string   `json:"uid"`
	DataCollectionName string   `json:"dataCollectionName"`
	FileInfo           FileInfo `json:"fileInfo"`
	// +optional
	ImportedTime metav1.Time        `json:"importedTime,omitempty"`
	Conditions   []common.Condition `json:"conditions,omitempty"`
}

var (
	InsertObject condition.Cond = "insertObject"
	DeleteObject condition.Cond = "deleteObject"
)
