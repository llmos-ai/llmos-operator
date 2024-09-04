package v1

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
	"github.com/llmos-ai/llmos-operator/pkg/utils/condition"
)

var (
	ClusterRoleExists    condition.Cond = "ClusterRoleExists"
	NamespacedRoleExists condition.Cond = "NamespacedRoleExists"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.summary"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

type GlobalRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired configs of the GlobalRole.
	// +optional
	Spec GlobalRoleTemplate `json:"spec,omitempty"`

	// Rules holds a list of PolicyRules that are applied to the local cluster only.
	// +optional
	Rules []rbacv1.PolicyRule `json:"rules,omitempty"`

	// NamespacedRules are the rules that are active in each namespace of this GlobalRole.
	// These are applied to the local cluster only.
	// * has no special meaning in the keys - these keys are read as raw strings
	// and must exactly match with one existing namespace.
	// +optional
	NamespacedRules map[string][]rbacv1.PolicyRule `json:"namespacedRules,omitempty"`

	// Status is the most recently observed status of the GlobalRole.
	// +optional
	Status GlobalRoleStatus `json:"status,omitempty"`
}

type GlobalRoleTemplate struct {
	// DisplayName is the human-readable name displayed in the UI.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// NewUserDefault specifies that all new users created should be bound to this role if true.
	// +optional
	NewUserDefault bool `json:"newUserDefault,omitempty"`

	// Builtin specifies that this GlobalRole was created by LLMOS if true.
	// +optional
	Builtin bool `json:"builtin,omitempty"`
}

type GlobalRoleStatus struct {
	// Conditions is a slice of Condition, indicating the status of specific backing RBAC objects.
	// There is one condition per ClusterRole and Role managed by the GlobalRole.
	// +optional
	Conditions []common.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the most recent generation (metadata.generation in GlobalRole CR)
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
