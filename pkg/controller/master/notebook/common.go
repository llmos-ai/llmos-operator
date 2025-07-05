package notebook

import (
	"crypto/sha256"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/llmos-ai/llmos-operator/pkg/apis/common"
	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	ctlsnapshotv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/snapshot.storage.k8s.io/v1"
	"github.com/llmos-ai/llmos-operator/pkg/utils/reconcilehelper"
)

const (
	NamePrefix     = "notebook-"
	DefaultFSGroup = int64(100)
)

func constructNoteBookStatefulSet(
	notebook *mlv1.Notebook,
	datasetVersionCache ctlmlv1.DatasetVersionCache,
	volumeSnapshotCache ctlsnapshotv1.VolumeSnapshotCache,
) (*v1.StatefulSet, error) {
	replicas := notebook.Spec.Replicas
	if metav1.HasAnnotation(notebook.ObjectMeta, constant.AnnotationResourceStopped) {
		replicas = 0
	}

	selector := GetNotebookSelector(notebook)
	ss := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getFormattedNotebookName(notebook),
			Namespace: notebook.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(notebook, notebook.GroupVersionKind()),
			},
			Labels: selector.MatchLabels,
		},
		Spec: v1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selector.MatchLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      selector.MatchLabels,
					Annotations: map[string]string{},
				},
				Spec: *notebook.Spec.Template.Spec.DeepCopy(),
			},
			VolumeClaimTemplates: reconcilehelper.CopyVolumeClaimTemplates(notebook.Spec.VolumeClaimTemplates),
		},
	}

	// copy all the notebook labels to the pod including pod default related labels
	l := &ss.Spec.Template.Labels
	for k, v := range notebook.Labels {
		(*l)[k] = v
	}

	// copy all the notebook annotations to the pod.
	a := &ss.Spec.Template.Annotations
	for k, v := range notebook.Annotations {
		if !strings.Contains(k, "kubectl") && !strings.Contains(k, "notebook") {
			(*a)[k] = v
		}
	}

	podSpec := &ss.Spec.Template.Spec
	container := &podSpec.Containers[0]
	container.Name = notebook.Name
	if container.WorkingDir == "" {
		container.WorkingDir = "/home/jovyan"
	}
	if container.Ports == nil {
		container.Ports = []corev1.ContainerPort{
			{
				ContainerPort: DefaultContainerPort,
				Name:          "notebook-port",
				Protocol:      "TCP",
			},
		}
	}

	if value, exists := os.LookupEnv("ADD_FSGROUP"); !exists || value == "true" {
		if podSpec.SecurityContext == nil {
			fsGroup := DefaultFSGroup
			podSpec.SecurityContext = &corev1.PodSecurityContext{
				FSGroup: &fsGroup,
			}
		}
	}

	// Handle dataset mountings
	if err := addDatasetMountings(ss, notebook, datasetVersionCache, volumeSnapshotCache); err != nil {
		return nil, fmt.Errorf("failed to add dataset mountings: %w", err)
	}

	return ss, nil
}
func getNotebookService(notebook *mlv1.Notebook) *corev1.Service {
	svcType := corev1.ServiceTypeClusterIP
	if notebook.Spec.ServiceType != "" {
		svcType = notebook.Spec.ServiceType
	}

	selector := GetNotebookSelector(notebook)
	// Define the desired Service object
	port := DefaultContainerPort
	containerPorts := notebook.Spec.Template.Spec.Containers[0].Ports
	if containerPorts != nil {
		port = containerPorts[0].ContainerPort
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getFormattedNotebookName(notebook),
			Namespace: notebook.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(notebook, notebook.GroupVersionKind()),
			},
			Labels: selector.MatchLabels,
		},
		Spec: corev1.ServiceSpec{
			Type:     svcType,
			Selector: selector.MatchLabels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       DefaultServingPort,
					TargetPort: intstr.FromInt32(port),
					Protocol:   "TCP",
				},
			},
		},
	}
	return svc
}

func getNotebookStatus(ss *v1.StatefulSet, pod *corev1.Pod) mlv1.NotebookStatus {
	status := mlv1.NotebookStatus{
		Conditions:     make([]common.Condition, 0),
		ReadyReplicas:  ss.Status.ReadyReplicas,
		ContainerState: corev1.ContainerState{},
		State:          "",
	}

	if reflect.DeepEqual(pod.Status, corev1.PodStatus{}) {
		logrus.Infof("notebook pod status is empty, skip updating conditions and state")
		return status
	}

	if len(pod.Status.ContainerStatuses) > 0 {
		cState := pod.Status.ContainerStatuses[0].State
		status.ContainerState = cState
		if cState.Running != nil {
			status.State = "Running"
		} else if cState.Waiting != nil {
			status.State = "Waiting"
		} else if cState.Terminated != nil {
			status.State = "Terminated"
		} else {
			status.State = "Unknown"
		}
	}

	// Mirror the pod conditions to the ModelService conditions
	for i := range pod.Status.Conditions {
		condition := reconcilehelper.PodCondToCond(pod.Status.Conditions[i])
		status.Conditions = append(status.Conditions, condition)
	}

	return status
}

func GetNotebookSelector(notebook *mlv1.Notebook) *metav1.LabelSelector {
	if notebook.Spec.Selector != nil {
		selector := notebook.Spec.Selector.DeepCopy()
		if selector.MatchLabels == nil {
			selector.MatchLabels = make(map[string]string)
		}
		selector.MatchLabels[constant.LabelLLMOSMLAppName] = strings.ToLower(notebook.Kind)
		selector.MatchLabels[constant.LabelNotebookName] = notebook.Name
		return selector
	}
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			constant.LabelLLMOSMLAppName: strings.ToLower(notebook.Kind),
			constant.LabelNotebookName:   notebook.Name,
		},
	}
}

func getFormattedNotebookName(notebook *mlv1.Notebook) string {
	return fmt.Sprintf("%s%s", NamePrefix, notebook.Name)
}

func getNotebookPodName(statefulSetName string) string {
	return fmt.Sprintf("%s-0", statefulSetName)
}

// addDatasetMountings adds dataset mountings to the StatefulSet
func addDatasetMountings(
	ss *v1.StatefulSet,
	notebook *mlv1.Notebook,
	datasetVersionCache ctlmlv1.DatasetVersionCache,
	volumeSnapshotCache ctlsnapshotv1.VolumeSnapshotCache,
) error {
	if len(notebook.Spec.DatasetMountings) == 0 {
		return nil
	}

	for i, mounting := range notebook.Spec.DatasetMountings {
		// Find the DatasetVersion by iterating through all DatasetVersions in the namespace
		datasetVersions, err := datasetVersionCache.List(notebook.Namespace, labels.SelectorFromSet(map[string]string{
			constant.LabelDatasetName:    mounting.DatasetName,
			constant.LabelDatasetVersion: mounting.Version,
		}))
		if err != nil {
			return fmt.Errorf("failed to list dataset versions: %w", err)
		}
		if len(datasetVersions) != 1 {
			logrus.Warnf("found %d dataset version from %s/%s", len(datasetVersions), mounting.DatasetName, mounting.Version)
			continue
		}

		datasetVersion := datasetVersions[0]
		// Check if the dataset version is published and has a volume snapshot
		if !datasetVersion.Spec.Publish || datasetVersion.Status.PublishStatus.Phase != mlv1.SnapshottingPhaseSnapshotReady {
			return fmt.Errorf("dataset version %s-%s is not published", mounting.DatasetName, mounting.Version)
		}

		snapshotName := datasetVersion.Status.PublishStatus.SnapshotName
		if snapshotName == "" {
			return fmt.Errorf("dataset version %s-%s does not have a volume snapshot", mounting.DatasetName, mounting.Version)
		}

		volumeSnapshot, err := volumeSnapshotCache.Get(datasetVersion.Namespace, snapshotName)
		if err != nil {
			return fmt.Errorf("failed to get volume snapshot %s: %w", snapshotName, err)
		}

		// Create PVC name for this dataset mounting
		pvcName := generatePVCName(notebook.Name, i, mounting.DatasetName, mounting.Version)

		// Create PVC with VolumeSnapshot as data source
		pvc := corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvcName,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: notebook.APIVersion,
						Kind:       notebook.Kind,
						Name:       notebook.Name,
						UID:        notebook.UID,
					},
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany},
				StorageClassName: ptr.To("llmos-ceph-block"),
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: *volumeSnapshot.Status.RestoreSize,
					},
				},
				DataSource: &corev1.TypedLocalObjectReference{
					Kind:     "VolumeSnapshot",
					Name:     datasetVersion.Status.PublishStatus.SnapshotName,
					APIGroup: ptr.To("snapshot.storage.k8s.io"),
				},
			},
		}

		// Add PVC to VolumeClaimTemplates
		ss.Spec.VolumeClaimTemplates = append(ss.Spec.VolumeClaimTemplates, pvc)

		// Add volume mount to the container
		volumeMount := corev1.VolumeMount{
			Name:      pvcName,
			MountPath: mounting.MountPath,
			ReadOnly:  true,
		}

		// Add volume mount to the first container (assuming it's the main notebook container)
		if len(ss.Spec.Template.Spec.Containers) > 0 {
			ss.Spec.Template.Spec.Containers[0].VolumeMounts = append(
				ss.Spec.Template.Spec.Containers[0].VolumeMounts,
				volumeMount,
			)
		}
	}

	return nil
}

// generatePVCName creates a PVC name that respects Kubernetes naming constraints
// Kubernetes resource names must be no more than 63 characters and follow DNS naming conventions
func generatePVCName(notebookName string, index int, datasetName, version string) string {
	// Create the base name
	baseName := fmt.Sprintf("%s-%d-%s-%s", notebookName, index, datasetName, version)

	// If the name is within the limit, return it as is
	if len(baseName) <= 63 {
		return baseName
	}

	// If too long, we need to truncate while maintaining uniqueness
	// Keep the index and a hash of the original components for uniqueness
	hashInput := fmt.Sprintf("%s-%s-%s", notebookName, datasetName, version)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(hashInput)))

	// Use first 8 characters of hash for uniqueness
	hashSuffix := hash[:8]

	// Calculate available space: 63 - len("-") - len(index) - len("-") - len(hashSuffix)
	indexStr := fmt.Sprintf("%d", index)
	availableSpace := 63 - 1 - len(indexStr) - 1 - len(hashSuffix)

	// Truncate notebook name to fit
	truncatedNotebook := notebookName
	if len(truncatedNotebook) > availableSpace {
		truncatedNotebook = truncatedNotebook[:availableSpace]
	}

	// Remove trailing hyphens
	truncatedNotebook = strings.TrimSuffix(truncatedNotebook, "-")

	return fmt.Sprintf("%s-%s-%s", truncatedNotebook, indexStr, hashSuffix)
}
