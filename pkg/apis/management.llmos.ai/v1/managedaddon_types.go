package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
	cond "github.com/llmos-ai/llmos-operator/pkg/utils/condition"
)

const (
	AddonCondChartDeployed cond.Cond = "ChartDeployed"
	AddonCondInProgress    cond.Cond = "InProgress"
	AddonCondReady         cond.Cond = "Ready"

	AddonStateEnabled    AddonState = "Enabled"
	AddonStateDisabled   AddonState = "Disabled"
	AddonStateDeployed   AddonState = "Deployed"
	AddonStateInProgress AddonState = "InProgress"
	AddonStateComplete   AddonState = "Complete"
	AddonStateError      AddonState = "Error"
	AddonStateFailed     AddonState = "Failed"
)

type AddonState string

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Repo",type=string,JSONPath=`.spec.repo`
// +kubebuilder:printcolumn:name="Chart",type=string,JSONPath=`.spec.chart`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Enabled",type=boolean,JSONPath=`.spec.enabled`

// ManagedAddon helps to manage the lifecycle of the LLMOS system addons.
type ManagedAddon struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ManagedAddonSpec   `json:"spec"`
	Status            ManagedAddonStatus `json:"status,omitempty"`
}

type ManagedAddonSpec struct {
	// +kubebuilder:validation:Required
	Repo string `json:"repo"`
	// +kubebuilder:validation:Required
	Chart string `json:"chart"`
	// +kubebuilder:validation:Required
	Version string `json:"version"`
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`
	// +optional
	ValuesContent string `json:"valuesContent,omitempty"`
}

type ManagedAddonStatus struct {
	// Conditions is an array of current conditions
	Conditions []common.Condition `json:"conditions,omitempty"`
	// State is the state of managedAddon.
	State AddonState `json:"state,omitempty"`

	// Represents time when the job was completed. It is not guaranteed to
	// be set in happens-before order across separate operations.
	// It is represented in RFC3339 form and is in UTC.
	// The completion time is set when the job finishes successfully, and only then.
	// The value cannot be updated or removed. The value indicates the same or
	// later point in time as the startTime field.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// The number of pods which reached phase Succeeded.
	// The value increases monotonically for a given spec. However, it may
	// decrease in reaction to scale down of elastic indexed jobs.
	// +optional
	Succeeded int32 `json:"succeeded,omitempty"`

	// The number of pods which have a Ready condition.
	// +optional
	Ready *int32 `json:"ready,omitempty"`

	// The number of pending and running pods which are not terminating (without
	// a deletionTimestamp).
	// The value is zero for finished jobs.
	// +optional
	Active int32 `json:"active,omitempty"`

	// The name of the job that was created for this managedAddon.
	JobName string `json:"jobName,omitempty"`
}
