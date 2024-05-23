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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/llmos-ai/llmos-controller/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-controller/pkg/utils/condition"
)

var (
	ModelFileCreated   condition.Cond = "Created"
	ModelFileCompleted condition.Cond = "Completed"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Model",type="string",JSONPath=`.status.model`
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.modelID"
// +kubebuilder:printcolumn:name="Size",type="string",JSONPath=`.status.byteSize`
// +kubebuilder:printcolumn:name="Model_Modified",type="date",JSONPath=".status.modifiedAt"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ModelFile is the Schema for the ModelFiles API
type ModelFile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModelFileSpec   `json:"spec,omitempty"`
	Status ModelFileStatus `json:"status,omitempty"`
}

// ModelFileSpec defines the desired state of ModelFile
type ModelFileSpec struct {
	// +kubebuilder:validation:Required
	FileSpec string `json:"fileSpec"`
	// +optional
	TagName string `json:"tagName,omitempty"`
	// +optional
	Description string `json:"description,omitempty"`
	// +optional
	PromptSuggestions []string `json:"promptSuggestions,omitempty"`
	// +optional
	Categories []string `json:"categories,omitempty"`
	// +optional
	// +kubebuilder:default:=true
	IsPublic bool `json:"isPublic,omitempty"`
}

// ModelFileStatus defines the observed state of ModelFile
type ModelFileStatus struct {
	Conditions []v1.Condition `json:"conditions,omitempty"`
	IsPublic   bool           `json:"isPublic,omitempty"`
	Model      string         `json:"model,omitempty"`
	ByteSize   string         `json:"byteSize,omitempty"`
	Size       int64          `json:"size,omitempty"`
	Digest     string         `json:"digest,omitempty"`
	ModelID    string         `json:"modelID,omitempty"`
	Details    ModelDetails   `json:"details,omitempty"`
	ModifiedAt string         `json:"modifiedAt,omitempty"`
	ExpiresAt  string         `json:"expiresAt,omitempty"`
}

type ModelDetails struct {
	ParentModel       string   `json:"parentModel,omitempty"`
	Format            string   `json:"format,omitempty"`
	Family            string   `json:"family,omitempty"`
	Families          []string `json:"families,omitempty"`
	ParameterSize     string   `json:"parameterSize,omitempty"`
	QuantizationLevel string   `json:"quantizationLevel,omitempty"`
}
