package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=nb,scope=Namespaced
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=`.metadata.labels['ml.llmos.ai\/notebook-type']`
// +kubebuilder:printcolumn:name="Cpu",type="string",JSONPath=`.spec.template.spec.containers[0].resources.limits.cpu`
// +kubebuilder:printcolumn:name="Memory",type="string",JSONPath=`.spec.template.spec.containers[0].resources.limits.memory`
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=`.metadata.creationTimestamp`

// Notebook is the Schema for the notebooks API
type Notebook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotebookSpec   `json:"spec,omitempty"`
	Status NotebookStatus `json:"status,omitempty"`
}

// NotebookSpec defines the desired state of Dataset
type NotebookSpec struct {
	// +optional, template describes the notebooks that will be created.
	Template NotebookTemplateSpec `json:"template,omitempty"`

	// +kubebuilder:validation:Required
	Replicas int32 `json:"replicas"`

	// selector is a label query over pods that should match the replica count.
	// It must match the pod template's labels.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// +optional, serviceType is the type of service that will be created.
	// +kubebuilder:default:=ClusterIP
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// +optional, list of PersistentVolumeClaims that will be created.
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`

	// +optional
	DatasetMountings []DatasetMounting `json:"datasetMountings,omitempty"`
}

type NotebookTemplateSpec struct {
	Spec corev1.PodSpec `json:"spec,omitempty"`
}

type DatasetMounting struct {
	DatasetName string `json:"datasetName"`
	Version     string `json:"version"`
	MountPath   string `json:"mountPath"`
}

// NotebookStatus defines the observed state of Dataset
type NotebookStatus struct {
	// Conditions is an array of current conditions
	Conditions []common.Condition `json:"conditions"`
	// ReadyReplicas is the number of Pods created by the StatefulSet controller that have a Ready Condition.
	ReadyReplicas int32 `json:"readyReplicas"`
	// ContainerState is the mirror state of underlying container
	ContainerState corev1.ContainerState `json:"containerState,omitempty"`
	// State is the state of the notebook
	State string `json:"state,omitempty"`
}
