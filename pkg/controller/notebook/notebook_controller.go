package notebook

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	ctlappsv1 "github.com/rancher/wrangler/v2/pkg/generated/controllers/apps/v1"
	ctlcorev1 "github.com/rancher/wrangler/v2/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v2/pkg/relatedresource"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"

	mgmtv1 "github.com/llmos-ai/llmos-controller/pkg/apis/management.llmos.ai/v1"
	mlv1 "github.com/llmos-ai/llmos-controller/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-controller/pkg/constant"
	ctlmlv1 "github.com/llmos-ai/llmos-controller/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-controller/pkg/server/config"
	"github.com/llmos-ai/llmos-controller/pkg/utils"
)

// Note: this is referred to the kubeflow notebook controller
// https://github.com/kubeflow/kubeflow/tree/master/components/notebook-controller

const (
	DefaultContainerPort = int32(8888)
	DefaultServingPort   = 80

	// DefaultFSGroup The default fsGroup of PodSecurityContext.
	// https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#podsecuritycontext-v1-core
	DefaultFSGroup = int64(100)
)

type Handler struct {
	scheme           *runtime.Scheme
	notebooks        ctlmlv1.NotebookClient
	statefulSets     ctlappsv1.StatefulSetClient
	statefulSetCache ctlappsv1.StatefulSetCache
	services         ctlcorev1.ServiceClient
	serviceCache     ctlcorev1.ServiceCache
	podCache         ctlcorev1.PodCache
	pvcHandler       *utils.PVCHandler
}

const (
	notebookControllerOnChange  = "notebook.onChange"
	notebookControllerCreatePVC = "notebook.createNoteBookPVC"
	notebookControllerWatchPods = "notebook.watchPods"
)

func Register(ctx context.Context, mgmt *config.Management) error {
	notebooks := mgmt.LLMFactory.Ml().V1().Notebook()
	ss := mgmt.AppsFactory.Apps().V1().StatefulSet()
	services := mgmt.CoreFactory.Core().V1().Service()
	pods := mgmt.CoreFactory.Core().V1().Pod()
	pvcs := mgmt.CoreFactory.Core().V1().PersistentVolumeClaim()
	h := Handler{
		scheme:           mgmt.Scheme,
		notebooks:        notebooks,
		statefulSets:     ss,
		statefulSetCache: ss.Cache(),
		services:         services,
		serviceCache:     services.Cache(),
		podCache:         pods.Cache(),
		pvcHandler:       utils.NewPVCHandler(pvcs, pvcs.Cache()),
	}

	notebooks.OnChange(ctx, notebookControllerOnChange, h.OnChanged)
	notebooks.OnChange(ctx, notebookControllerCreatePVC, h.createNoteBookPVC)
	relatedresource.Watch(ctx, notebookControllerWatchPods, h.ReconcileNotebookPodOwners, notebooks, pods)
	return nil
}

func (h *Handler) OnChanged(_ string, notebook *mlv1.Notebook) (*mlv1.Notebook, error) {
	if notebook == nil || notebook.DeletionTimestamp != nil {
		return nil, nil
	}

	// create notebook statefulSet if not exist
	ss, err := h.ensureStatefulSet(notebook)
	if err != nil {
		return notebook, err
	}

	// sync pod template spec from ss to notebook after update
	if !reflect.DeepEqual(ss.Spec.Template.Spec, notebook.Spec.Template.Spec) {
		nbCpy := notebook.DeepCopy()
		nbCpy.Spec.Template.Spec = ss.Spec.Template.Spec
		if _, err = h.notebooks.Update(nbCpy); err != nil {
			return notebook, err
		}
	}

	// create notebook service if not exist
	if err = h.generateService(notebook); err != nil {
		return notebook, err
	}

	// update notebook status
	err = h.updateNotebookStatus(notebook, ss)
	if err != nil {
		return notebook, err
	}

	return notebook, nil
}

func (h *Handler) ensureStatefulSet(notebook *mlv1.Notebook) (*v1.StatefulSet, error) {
	logrus.Debugf("Ensure statefulset for notebook %s/%s", notebook.Namespace, notebook.Name)
	ss, err := h.statefulSetCache.Get(notebook.Namespace, notebook.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			logrus.Infof("Generating statefulset for notebook %s/%s", notebook.Namespace, notebook.Name)
			ss = getNoteBookStatefulSet(notebook)

			if err = ctrl.SetControllerReference(notebook, ss, h.scheme); err != nil {
				return nil, err
			}
			return h.statefulSets.Create(ss)
		}
		return nil, err
	}

	if !reflect.DeepEqual(notebook.Spec.Template.Spec, ss.Spec.Template.Spec) {
		logrus.Infof("Updating notebook statefulset %s/%s", notebook.Namespace, notebook.Name)
		ssCopy := ss.DeepCopy()
		ssCopy.Spec.Template.Spec = notebook.Spec.Template.Spec
		if ss, err = h.statefulSets.Update(ssCopy); err != nil {
			return ss, err
		}
	}

	return ss, nil
}

func getNoteBookStatefulSet(notebook *mlv1.Notebook) *v1.StatefulSet {
	replicas := int32(1)
	if metav1.HasAnnotation(notebook.ObjectMeta, constant.AnnotationResourceStopped) {
		replicas = 0
	}

	ss := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      notebook.Name,
			Namespace: notebook.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: notebook.APIVersion,
					Kind:       notebook.Kind,
					Name:       notebook.Name,
					UID:        notebook.UID,
				},
			},
		},
		Spec: v1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: getNotebookPodLabel(notebook),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      getNotebookPodLabel(notebook),
					Annotations: map[string]string{},
				},
				Spec: *notebook.Spec.Template.Spec.DeepCopy(),
			},
		},
	}

	// copy all the notebook labels to the pod including pod default related labels
	l := &ss.Spec.Template.ObjectMeta.Labels
	for k, v := range notebook.ObjectMeta.Labels {
		(*l)[k] = v
	}

	// copy all the notebook annotations to the pod.
	a := &ss.Spec.Template.ObjectMeta.Annotations
	for k, v := range notebook.ObjectMeta.Annotations {
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

	// For some platforms (like OpenShift), adding fsGroup: 100 is troublesome.
	// This allows for those platforms to bypass the automatic addition of the fsGroup
	// and will allow for the Pod Security Policy controller to make an appropriate choice
	// https://github.com/kubernetes-sigs/controller-runtime/issues/4617
	if value, exists := os.LookupEnv("ADD_FSGROUP"); !exists || value == "true" {
		if podSpec.SecurityContext == nil {
			fsGroup := DefaultFSGroup
			podSpec.SecurityContext = &corev1.PodSecurityContext{
				FSGroup: &fsGroup,
			}
		}
	}
	return ss
}

func (h *Handler) generateService(notebook *mlv1.Notebook) error {
	svcName := fmt.Sprintf("%s-notebook", notebook.Name)
	svc, err := h.serviceCache.Get(notebook.Namespace, svcName)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if svc == nil {
		logrus.Infof("Creating new service %s/%s", notebook.Namespace, svcName)
		svc = getService(notebook, svcName)

		if err = ctrl.SetControllerReference(notebook, svc, h.scheme); err != nil {
			return err
		}

		if _, err = h.services.Create(svc); err != nil {
			return err
		}
	}

	if svc.Spec.Type != notebook.Spec.ServiceType {
		svcCpy := svc.DeepCopy()
		svcCpy.Spec.Type = notebook.Spec.ServiceType
		if _, err = h.services.Update(svcCpy); err != nil {
			return err
		}
	}

	return nil
}

func getService(notebook *mlv1.Notebook, svcName string) *corev1.Service {
	svcType := corev1.ServiceTypeClusterIP
	if notebook.Spec.ServiceType != "" {
		svcType = notebook.Spec.ServiceType
	}
	// Define the desired Service object
	port := DefaultContainerPort
	containerPorts := notebook.Spec.Template.Spec.Containers[0].Ports
	if containerPorts != nil {
		port = containerPorts[0].ContainerPort
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: notebook.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:     svcType,
			Selector: getNotebookPodLabel(notebook),
			Ports: []corev1.ServicePort{
				{
					Name:       "http-" + notebook.Name,
					Port:       DefaultServingPort,
					TargetPort: intstr.FromInt32(port),
					Protocol:   "TCP",
				},
			},
		},
	}
	return svc
}

func getNotebookPodLabel(notebook *mlv1.Notebook) map[string]string {
	return map[string]string{
		"statefulset":     notebook.Name,
		notebookNameLabel: notebook.Name,
	}
}

func (h *Handler) updateNotebookStatus(notebook *mlv1.Notebook, ss *v1.StatefulSet) error {
	toUpdateStatus := false
	pod, err := h.podCache.Get(ss.Namespace, fmt.Sprintf("%s-0", ss.Name))
	if err != nil {
		if errors.IsNotFound(err) {
			logrus.Infof("Notebook pod not found: %s, skipp updating and waiting for reconcile", err)
			return nil
		}
		return err
	}

	if reflect.DeepEqual(pod.Status, corev1.PodStatus{}) {
		logrus.Debugln("Empty pod status, won't update notebook status")
		return nil
	}

	nbCopy := notebook.DeepCopy()
	if ss.Status.ReadyReplicas != nbCopy.Status.ReadyReplicas {
		toUpdateStatus = true
	}
	status := mlv1.NotebookStatus{
		Conditions:    make([]mgmtv1.Condition, 0),
		ReadyReplicas: ss.Status.ReadyReplicas,
		State:         corev1.ContainerState{},
	}

	// Update status of the CR using the ContainerState of
	// the container that has the same name as the CR.
	// If no container of same name is found, the state of the CR is not updated.
	logrus.Debugln("Calculating Notebook's containerState")
	notebookContainerFound := false
	for i := range pod.Status.ContainerStatuses {
		if !strings.Contains(pod.Status.ContainerStatuses[i].Name, notebook.Name) {
			continue
		}

		if pod.Status.ContainerStatuses[i].State == notebook.Status.State {
			continue
		}

		// Update Notebook CR's status.ContainerState
		cs := pod.Status.ContainerStatuses[i].State
		logrus.Debugf("Updating notebook cr state: %s", cs)

		status.State = cs
		notebookContainerFound = true
		break
	}

	if !notebookContainerFound {
		logrus.Debugf("Could not find notebook container %s, Will not update notebook's status.state", notebook.Name)
	}

	// Mirroring pod condition
	notebookConditions := make([]mgmtv1.Condition, 0)
	for i := range pod.Status.Conditions {
		condition := PodCondToNotebookCond(pod.Status.Conditions[i])
		notebookConditions = append(notebookConditions, condition)
	}
	// update status
	status.Conditions = notebookConditions

	if !reflect.DeepEqual(status.Conditions, notebookConditions) {
		toUpdateStatus = true
	}

	if !reflect.DeepEqual(status.State, nbCopy.Status.State) {
		toUpdateStatus = true
	}

	if toUpdateStatus {
		nbCopy.Status = status
		_, err = h.notebooks.UpdateStatus(nbCopy)
	}

	return err
}
