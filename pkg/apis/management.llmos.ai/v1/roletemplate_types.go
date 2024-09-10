package v1

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

type RoleTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired configs of the RoleTemplate.
	// +optional
	Spec RoleTemplateSpec `json:"spec,omitempty"`

	// Rules hold a list of PolicyRules for this RoleTemplate.
	// +optional
	Rules []rbacv1.PolicyRule `json:"rules,omitempty"`

	// Status is the most recently observed status of the RoleTemplate.
	// +optional
	Status RoleTemplateStatus `json:"status,omitempty"`
}

type RoleTemplateSpec struct {
	// DisplayName is the human-readable name displayed in the UI.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Builtin specifies that this RoleTemplate was created by LLMOS if true.
	// +optional
	Builtin bool `json:"builtin,omitempty"`

	// Locked specified that if new bindings will not be able to use this RoleTemplate.
	// +optional
	Locked bool `json:"locked,omitempty"`

	// NewNamespaceDefault specifies that this RoleTemplate should be applied to all new created namespaces if true.
	NewNamespaceDefault bool `json:"newNamespaceDefault,omitempty"`
}

type RoleTemplateStatus struct {
	// Conditions is a slice of Condition, indicating the status of specific backing RBAC objects.
	// There is one condition per ClusterRole and Role managed by the RoleTemplate.
	// +optional
	Conditions []common.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the most recent generation (metadata.generation in RoleTemplate CR)
	// observed by the controller. Populated by the system.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastUpdate is a k8s timestamp of the last time the status was updated.
	// +optional
	LastUpdate string `json:"lastUpdateTime,omitempty"`

	// State represent a state of "Complete", "InProgress" or "Error".
	// +optional
	// +kubebuilder:validation:Enum={"Complete","InProgress","Error"}
	State string `json:"state,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:resource:scope=Cluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RoleTemplateBinding binds a given subject user to a GlobalRole.
type RoleTemplateBinding struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// RoleTemplateRef can reference a RoleTemplate in the current namespace or a GlobalRole in the global namespace.
	// If the RoleTemplateRef cannot be resolved, the Webhook must return an error.
	// This field is immutable.
	RoleTemplateRef RoleTemplateRef `json:"roleTemplateRef"`
	// Subjects holds references to the objects the global role applies to.
	// +optional
	// +listType=atomic
	Subjects []rbacv1.Subject `json:"subjects,omitempty"`

	// NamespaceId is the namespace id of the namespace the role template binding is applied to.
	NamespaceId string `json:"namespaceId,omitempty"`
}

type RoleTemplateRef struct {
	// APIGroup is the group for the resource being referenced
	APIGroup string `json:"apiGroup"`
	// Kind is the type of resource being referenced
	Kind string `json:"kind"`
	// Name is the name of resource being referenced
	Name string `json:"name"`
}
