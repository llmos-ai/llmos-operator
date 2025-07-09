package localmodel

import (
	"context"
	"fmt"
	"path"
	"sync"

	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/ptr"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/common/snapshotting"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
)

const (
	localModelOnChangeName        = "localModel.OnChange"
	localModelVersionOnChangeName = "localModelVersion.OnChange"
	localModelVersionOnRemoveName = "localModelVersion.OnRemove"

	volumeName      = "model-volume"
	volumeMountPath = "/root/"
)

type handler struct {
	ctx context.Context

	LocalModelClient        ctlmlv1.LocalModelClient
	LocalModelCache         ctlmlv1.LocalModelCache
	LocalModelVersionClient ctlmlv1.LocalModelVersionClient
	LocalModelVersionCache  ctlmlv1.LocalModelVersionCache
	RegistryClient          ctlmlv1.RegistryClient
	RegistryCache           ctlmlv1.RegistryCache
	ModelClient             ctlmlv1.ModelClient
	ModelCache              ctlmlv1.ModelCache
	PVCClient               ctlcorev1.PersistentVolumeClaimClient
	PVCCache                ctlcorev1.PersistentVolumeClaimCache

	rm                  *registry.Manager
	snapshottingManager *snapshotting.Manager
	sync.Mutex
}

func Register(ctx context.Context, mgmt *config.Management, _ config.Options) error {
	localModels := mgmt.LLMFactory.Ml().V1().LocalModel()
	localModelVersions := mgmt.LLMFactory.Ml().V1().LocalModelVersion()
	registries := mgmt.LLMFactory.Ml().V1().Registry()
	models := mgmt.LLMFactory.Ml().V1().Model()
	pvcs := mgmt.CoreFactory.Core().V1().PersistentVolumeClaim()
	secrets := mgmt.CoreFactory.Core().V1().Secret()

	h := handler{
		ctx:                     mgmt.Ctx,
		LocalModelClient:        localModels,
		LocalModelCache:         localModels.Cache(),
		LocalModelVersionClient: localModelVersions,
		LocalModelVersionCache:  localModelVersions.Cache(),
		RegistryClient:          registries,
		RegistryCache:           registries.Cache(),
		ModelClient:             models,
		ModelCache:              models.Cache(),
		PVCClient:               pvcs,
		PVCCache:                pvcs.Cache(),
	}

	// Create snapshotting manager with handler as ResourceHandler
	snapshottingMgr, err := snapshotting.NewManager(mgmt, &h)
	if err != nil {
		return fmt.Errorf("failed to create snapshotting manager: %w", err)
	}

	// Set snapshotting manager in handler
	h.snapshottingManager = snapshottingMgr

	h.rm = registry.NewManager(secrets.Cache().Get, registries.Cache().Get)

	localModels.OnChange(mgmt.Ctx, localModelOnChangeName, h.OnChange)
	localModelVersions.OnChange(mgmt.Ctx, localModelVersionOnChangeName, h.OnChangeVersion)
	localModelVersions.OnRemove(mgmt.Ctx, localModelVersionOnRemoveName, h.OnRemoveVersion)

	return nil
}

// OnChange to set the default version
func (h *handler) OnChange(_ string, localModel *mlv1.LocalModel) (*mlv1.LocalModel, error) {
	if localModel == nil || localModel.DeletionTimestamp != nil {
		return localModel, nil
	}
	ns, name, defaultVersion := localModel.Namespace, localModel.Name, localModel.Spec.DefaultVersion
	logrus.Debugf("set default version %s/%s to %s", ns, name, defaultVersion)

	if defaultVersion == "" {
		return h.setLatestAsDefaultVersion(localModel)
	}

	v, err := h.LocalModelVersionCache.Get(ns, defaultVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get the default version %s/%s of %s/%s: %w", ns, defaultVersion, ns, name, err)
	}
	// update the local model status
	if localModel.Status.DefaultVersion == v.Status.Version &&
		localModel.Status.DefaultVersionName == v.Name {
		return localModel, nil
	}
	localModelCopy := localModel.DeepCopy()
	localModelCopy.Status.DefaultVersion = v.Status.Version
	localModelCopy.Status.DefaultVersionName = v.Name
	mlv1.Ready.True(localModelCopy)
	return h.LocalModelClient.UpdateStatus(localModelCopy)
}

func (h *handler) OnChangeVersion(_ string, version *mlv1.LocalModelVersion) (*mlv1.LocalModelVersion, error) {
	if version == nil || version.DeletionTimestamp != nil {
		return version, nil
	}

	logrus.Infof("local model version %s/%s changed", version.Namespace, version.Name)

	v, err := h.assignVersion(version)
	if err != nil {
		return version, fmt.Errorf("failed to assign version to local model version %s/%s: %w",
			version.Namespace, version.Name, err)
	}

	// start snapshotting process if needed
	if err := h.doSnapshot(h.ctx, version); err != nil {
		return version, fmt.Errorf("failed to start snapshotting for local model version %s/%s: %w",
			version.Namespace, version.Name, err)
	}

	if err := h.setAsDefault(version); err != nil {
		return version, fmt.Errorf("failed to set latest version as default version: %w", err)
	}

	return v, nil
}

func (h *handler) OnRemoveVersion(_ string, version *mlv1.LocalModelVersion) (*mlv1.LocalModelVersion, error) {
	if version == nil || version.DeletionTimestamp == nil {
		return version, nil
	}

	lm, err := h.LocalModelCache.Get(version.Namespace, version.Spec.LocalModel)
	if err != nil {
		return version, fmt.Errorf("failed to get local model %s/%s: %w", version.Namespace, version.Spec.LocalModel, err)
	}
	// set default version to empty if the default version is deleted
	// and the localmodel controller will set the latest version as default version
	if lm.Spec.DefaultVersion == version.Name {
		lmCopy := lm.DeepCopy()
		lmCopy.Spec.DefaultVersion = ""
		if _, err := h.LocalModelClient.Update(lmCopy); err != nil {
			return version, fmt.Errorf("failed to set default version of %s/%s as empty: %w",
				version.Namespace, version.Spec.LocalModel, err)
		}
		logrus.Infof("set default version of %s/%s to empty since the original default version %s is deleted",
			version.Namespace, version.Spec.LocalModel, version.Name)
	}
	// reset the latest version as default version
	if lm.Spec.DefaultVersion == "" && lm.Status.DefaultVersionName == version.Name {
		if _, err := h.setLatestAsDefaultVersion(lm); err != nil {
			return version, fmt.Errorf("failed to set latest version as default version: %w", err)
		}
	}

	return nil, nil
}

func (h *handler) assignVersion(v *mlv1.LocalModelVersion) (*mlv1.LocalModelVersion, error) {
	if v.Status.Version != 0 {
		return v, nil
	}

	h.Lock()
	defer h.Unlock()
	// get the latest version of the local model
	var latestVersion int
	list, err := h.LocalModelVersionClient.List(v.Namespace, metav1.ListOptions{
		LabelSelector: labels.Set{constant.LabelLocalModelName: v.Spec.LocalModel}.AsSelector().String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list local model versions of %s: %w", v.Spec.LocalModel, err)
	}

	for _, item := range list.Items {
		if item.Status.Version > latestVersion {
			latestVersion = item.Status.Version
		}
	}

	// assign version to the local model version
	versionCopy := v.DeepCopy()
	versionCopy.Status.Version = latestVersion + 1
	return h.LocalModelVersionClient.UpdateStatus(versionCopy)
}

func (h *handler) doSnapshot(ctx context.Context, version *mlv1.LocalModelVersion) error {
	// Create the spec for snapshotting
	spec := &snapshotting.Spec{
		Namespace: version.Namespace,
		Name:      version.Name,
		Labels: map[string]string{
			constant.LabelLocalModelName: version.Spec.LocalModel,
		},
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: version.APIVersion,
				Kind:       version.Kind,
				Name:       version.Name,
				UID:        version.UID,
				Controller: ptr.To(true),
			},
		},
		PVCSpec: snapshotting.PVCSpec{
			AccessModes:               []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			RestoreFromLatestSnapshot: true,
		},
		JobSpec: snapshotting.JobSpec{
			BackoffLimit:            ptr.To(int32(1)),
			TTLSecondsAfterFinished: ptr.To(int32(86400)), // 24 hours
			Image:                   settings.ModelDownloaderImage.Get(),
			Args: []string{
				fmt.Sprintf("--name=%s/%s", version.Labels[constant.LabelModelNamespace], version.Labels[constant.LabelModelName]),
				fmt.Sprintf("--output-dir=%s", path.Join(volumeMountPath, "models", version.Namespace, version.Spec.LocalModel)),
				"--debug=true",
			},
		},
	}

	// Start the snapshotting process
	return h.snapshottingManager.DoSnapshot(ctx, spec)
}

func (h *handler) setAsDefault(v *mlv1.LocalModelVersion) error {
	if !mlv1.Ready.IsTrue(v) {
		return nil
	}

	localModel, err := h.LocalModelCache.Get(v.Namespace, v.Spec.LocalModel)
	if err != nil {
		return fmt.Errorf("failed to get local model %s/%s: %w", v.Namespace, v.Spec.LocalModel, err)
	}
	if localModel.Spec.DefaultVersion != "" {
		return nil
	}

	if v.Status.Version < localModel.Status.DefaultVersion {
		return nil
	}
	if v.Status.Version == localModel.Status.DefaultVersion &&
		v.Name == localModel.Status.DefaultVersionName {
		return nil
	}

	localModelCopy := localModel.DeepCopy()
	localModelCopy.Status.DefaultVersion = v.Status.Version
	localModelCopy.Status.DefaultVersionName = v.Name
	mlv1.Ready.True(localModelCopy)
	if _, err := h.LocalModelClient.UpdateStatus(localModelCopy); err != nil {
		return fmt.Errorf("failed to update status of local model %s/%s: %w", v.Namespace, v.Spec.LocalModel, err)
	}

	return nil
}

// If no default version is set, set the latest version as default version
func (h *handler) setLatestAsDefaultVersion(localModel *mlv1.LocalModel) (*mlv1.LocalModel, error) {
	if localModel.Spec.DefaultVersion != "" {
		return localModel, nil
	}

	namespace, name := localModel.Namespace, localModel.Name
	versions, err := h.LocalModelVersionCache.List(namespace, labels.Set{
		constant.LabelLocalModelName: name,
	}.AsSelector())
	if err != nil {
		return nil, fmt.Errorf("failed to list local model versions of %s/%s: %w", namespace, name, err)
	}

	localModelCopy := localModel.DeepCopy()
	localModelCopy.Status.DefaultVersion = 0
	localModelCopy.Status.DefaultVersionName = ""
	mlv1.Ready.False(localModelCopy)
	for _, v := range versions {
		if v.Status.Version > localModelCopy.Status.DefaultVersion &&
			v.DeletionTimestamp == nil && mlv1.Ready.IsTrue(v) {
			localModelCopy.Status.DefaultVersion = v.Status.Version
			localModelCopy.Status.DefaultVersionName = v.Name
			mlv1.Ready.True(localModelCopy)
		}
	}

	if localModelCopy.Status.DefaultVersion == localModel.Status.DefaultVersion &&
		localModelCopy.Status.DefaultVersionName == localModel.Status.DefaultVersionName {
		return localModel, nil
	}

	newLocalModel, err := h.LocalModelClient.UpdateStatus(localModelCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to update status of local model %s/%s: %w", namespace, name, err)
	}
	return newLocalModel, nil
}
