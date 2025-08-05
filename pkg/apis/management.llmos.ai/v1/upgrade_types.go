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

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
	"github.com/llmos-ai/llmos-operator/pkg/utils/condition"
)

var (
	// UpgradeCompleted is true when the upgrade is completion
	UpgradeCompleted condition.Cond = "Completed"
	// UpgradeChartsRepoReady is true when the upgrade chart repo is ready
	UpgradeChartsRepoReady condition.Cond = "ChartsRepoReady"
	// ManifestUpgradeComplete is true when the llmos-operator charts is upgraded
	ManifestUpgradeComplete condition.Cond = "ManifestUpgradeComplete"
	// ManagedAddonsIsReady is true when all the activated managed-addons are upgraded
	ManagedAddonsIsReady condition.Cond = "ManagedAddonsIsReady"
	// NodesUpgraded is true when all nodes are upgraded
	NodesUpgraded condition.Cond = "NodesUpgraded"
)

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=`.spec.version`
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Upgrade is the Schema for the upgrades API
type Upgrade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UpgradeSpec   `json:"spec,omitempty"`
	Status UpgradeStatus `json:"status,omitempty"`
}

// UpgradeSpec defines the desired state of Upgrade
type UpgradeSpec struct {
	// +kubebuilder:validation:Required
	// The LLMOS Operator version to upgrade to
	Version string `json:"version"`

	// +kubebuilder:validation:Required
	// Specify the Kubernetes version to upgrade to
	KubernetesVersion string `json:"kubernetesVersion"`

	// +Optional, override the default image registry if provided
	Registry string `json:"registry,omitempty"`

	// +Optional, Specify the drain spec
	Drain *upgradev1.DrainSpec `json:"drain,omitempty"`
}

// UpgradeStatus defines the observed state of Upgrade
type UpgradeStatus struct {
	// +optional
	Conditions []common.Condition `json:"conditions,omitempty"`
	// +optional, a map of node name to upgrade status
	NodeStatuses map[string]NodeUpgradeStatus `json:"nodeStatuses,omitempty"`
	// +optional, previous llmos version
	PreviousVersion string `json:"previousVersion,omitempty"`
	// +optional, previous Kubernetes version
	PreviousKubernetesVersion string `json:"PreviousKubernetesVersion,omitempty"`
	// +optional, Node image used for upgrade
	NodeImageId string `json:"nodeImageId,omitempty"`
	// +optional
	State string `json:"state,omitempty"`
	// +optional, save the applied version
	AppliedVersion string `json:"appliedVersion,omitempty"`
	// +optional, a list of upgrade jobs that are required to be completed before the upgrade can be ready
	UpgradeJobs []UpgradeJobStatus `json:"upgradeJobs,omitempty"`
	// +optional, a map of plan name to upgrade status
	PlanStatus []UpgradePlanStatus `json:"planStatus,omitempty"`
	// +optional, a list of managed addon upgrade status
	ManagedAddonStatus []UpgradeManagedAddonStatus `json:"managedAddonStatus,omitempty"`
	// +optional
	StartTime metav1.Time `json:"startTime,omitempty"`
	// +optional
	CompleteTime metav1.Time `json:"completeTime,omitempty"`
}

type NodeUpgradeStatus struct {
	// +optional
	State string `json:"state,omitempty"`
	// +optional
	Reason string `json:"reason,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
}

type UpgradePlanStatus struct {
	Name           string      `json:"name"`
	Complete       bool        `json:"complete"`
	LatestHash     string      `json:"latestHash,omitempty"`
	LatestVersion  string      `json:"latestVersion,omitempty"`
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

type UpgradeJobStatus struct {
	Name           string      `json:"name"`
	HelmChartName  string      `json:"helmChartName"`
	Complete       bool        `json:"complete"`
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

type UpgradeManagedAddonStatus struct {
	Name           string      `json:"name"`
	JobName        string      `json:"jobName,omitempty"`
	Complete       bool        `json:"complete"`
	Disabled       bool        `json:"disabled"`
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}
