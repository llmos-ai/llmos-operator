package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Username",type="string",JSONPath=`.spec.username`

type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              UserSpec   `json:"spec"`
	Status            UserStatus `json:"status,omitempty"`
}

type UserSpec struct {
	// +kubebuilder:validation:Required
	Username string `json:"username"`

	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// +optional
	Description string `json:"description,omitempty"`

	// +kubebuilder:validation:Required
	Password string `json:"password"`

	// +optional
	IsAdmin bool `json:"isAdmin"`

	// +kubebuilder:default:=true
	IsActive bool `json:"isActive"`
}

type UserStatus struct {
	Conditions     []common.Condition `json:"conditions,omitempty"`
	LastUpdateTime string             `json:"lastUpdateTime,omitempty"`
	IsAdmin        bool               `json:"isAdmin"`
	IsActive       bool               `json:"isActive"`
}
