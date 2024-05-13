/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/llmos-ai/llmos-controller/pkg/utils/condition"
)

var (
	UpgradeCompleted condition.Cond = "Completed"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Image",type="string",JSONPath=`.spec.image`

// Upgrade is the Schema for the upgrades API
type Upgrade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UpgradeSpec   `json:"spec,omitempty"`
	Status UpgradeStatus `json:"status,omitempty"`
}

// UpgradeSpec defines the desired state of Upgrade
type UpgradeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:validation:Required
	Version                  string `json:"version"`
	*upgradev1.ContainerSpec `json:",omitempty"`
	Drain                    *upgradev1.DrainSpec `json:"drain,omitempty"`
}

// UpgradeStatus defines the observed state of Upgrade
type UpgradeStatus struct {
	// +optional
	PreviousVersion string `json:"previousVersion,omitempty"`
	// +optional
	ImageID string `json:"imageID,omitempty"`
	// +optional
	NodeStatuses map[string]NodeUpgradeStatus `json:"nodeStatuses,omitempty"`
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`
	// +optional
	PlanStatus upgradev1.PlanStatus `json:"planStatus,omitempty"`
}

type NodeUpgradeStatus struct {
	// +optional
	State string `json:"state,omitempty"`
	// +optional
	Reason string `json:"reason,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
}
