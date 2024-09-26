package v1

import (
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Model",type="string",JSONPath=`.spec.model`
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ModelService is a deployment for the LLM Model
type ModelService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ModelServiceSpec   `json:"spec,omitempty"`
	Status ModelServiceStatus `json:"status,omitempty"`
}

// ModelServiceSpec defines the desired state of ModelFile
type ModelServiceSpec struct {
	// +kubebuilder:validation:Required
	ModelName string `json:"model"`

	// +optional, name of the model to serve in API
	ServedModelName string `json:"servedModelName,omitempty"`

	// +kubebuilder:validation:Required
	Replicas int32 `json:"replicas"`

	// selector is a label query over pods that should match the replica count.
	// It must match the pod template's labels.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// +kubebuilder:default:=ClusterIP
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// +optional, list of persistent volume claims
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`

	// +optional, modelService's statefulset update strategy
	UpdateStrategy v1.StatefulSetUpdateStrategy `json:"updateStrategy,omitempty"`

	// +optional, pod template spec of the model
	Template ModelServiceTemplateSpec `json:"template,omitempty"`

	// +optional, map of accelerator name to number of accelerators
	// e.g., 4090:2 means only schedule to a node with 2 4090 GPUs
	// TODO: support accelerators by node selection
	Accelerators map[string]uint8 `json:"accelerators,omitempty"`

	// +optional, enable gradio GUI of the model
	EnableGUI bool `json:"enableGUI,omitempty"`
}

type ModelServiceTemplateSpec struct {
	Spec corev1.PodSpec `json:"spec,omitempty"`
}

// ModelServiceStatus defines the observed state of ModelFile
type ModelServiceStatus struct {
	Conditions []common.Condition `json:"conditions,omitempty"`
	// ReadyReplicas is the number of Pods created by the controller that have a Ready Condition
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
	// ContainerState is the mirror state of underlying container
	ContainerState corev1.ContainerState `json:"containerState,omitempty"`
	// State is the state of the model service
	State string `json:"state,omitempty"`
}
