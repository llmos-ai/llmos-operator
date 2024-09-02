package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Description",type="string",JSONPath=`.metadata.annotations.field\.llmos\.io\/description`
// +kubebuilder:printcolumn:name="Expires",type="string",JSONPath=`.status.expiresAt`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

type Token struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TokenSpec   `json:"spec"`
	Status            TokenStatus `json:"status,omitempty"`
}

type TokenSpec struct {
	// +kubebuilder:validation:Required
	UserId string `json:"userId"`

	// +kubebuilder:validation:Required
	AuthProvider string `json:"authProvider"`

	// +optional
	Expired    bool   `json:"expired,omitempty"`
	TTLSeconds int64  `json:"ttlSeconds,omitempty"`
	Token      string `json:"token,omitempty"`
}

type TokenStatus struct {
	ExpiresAt metav1.Time `json:"expiresAt,omitempty"`
	IsExpired bool        `json:"isExpired"`
}
