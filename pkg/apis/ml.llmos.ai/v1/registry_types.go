package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
	"github.com/llmos-ai/llmos-operator/pkg/utils/condition"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:shortName=reg;regs,scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Backend",type="string",JSONPath=`.spec.backendType`
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=`.spec.url`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Registry is a cluster-level resource for managing model registries
type Registry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegistrySpec   `json:"spec,omitempty"`
	Status RegistryStatus `json:"status,omitempty"`
}

// RegistrySpec defines the desired state of Registry
type RegistrySpec struct {
	// BackendType is the type of backend storage (e.g., S3)
	BackendType string `json:"backendType"`
	// URL is the URL of the backend storage (e.g., S3 bucket address)
	URL string `json:"url"`
	// AccessCredentialSecretName is the name of the secret containing the access credentials
	AccessCredentialSecretName string `json:"accessCredentialSecretName"`
}

// RegistryStatus defines the observed state of Registry
type RegistryStatus struct {
	// Conditions is a list of conditions representing the status of the Registry
	Conditions []common.Condition `json:"conditions,omitempty"`
}

var (
	Accessible condition.Cond = "accessible"
)
