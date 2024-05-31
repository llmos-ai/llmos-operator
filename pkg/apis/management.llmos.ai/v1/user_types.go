package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=`.displayName`
// +kubebuilder:printcolumn:name="Username",type="string",JSONPath=`.username`

type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// +optional
	Description string `json:"description,omitempty"`

	// +kubebuilder:validation:Required
	Username string `json:"username"`

	// +kubebuilder:validation:Required
	Password string `json:"password"`

	// +optional
	IsAdmin bool `json:"isAdmin"`
}
