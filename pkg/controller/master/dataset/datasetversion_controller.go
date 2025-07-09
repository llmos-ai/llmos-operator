package dataset

import (
	"fmt"
	"path"
	"reflect"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/common/snapshotting"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/registry/backend"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
)

func (h *handler) OnChangeDatasetVersion(_ string, dv *mlv1.DatasetVersion) (*mlv1.DatasetVersion, error) {
	if dv == nil || dv.DeletionTimestamp != nil {
		return dv, nil
	}

	logrus.Infof("version %s(%s) of dataset %s/%s changed", dv.Spec.Version, dv.Name, dv.Namespace, dv.Spec.Dataset)

	dvCopy := dv.DeepCopy()

	// check dataset ready
	dataset, err := h.checkDatasetReady(dv.Namespace, dv.Spec.Dataset)
	if err != nil {
		return h.updateDatasetVersionStatus(dvCopy, dv, err)
	}
	dvCopy.Status.Registry = dataset.Spec.Registry

	if !mlv1.Ready.IsTrue(dv) {
		// create directory for dataset version
		b, err := h.rm.NewBackendFromRegistry(h.ctx, dataset.Spec.Registry)
		if err != nil {
			return h.updateDatasetVersionStatus(dvCopy, dv, fmt.Errorf(registry.ErrCreateBackendClient, err))
		}
		versionDir := path.Join(dataset.Status.RootPath, dv.Spec.Version)
		if err = b.CreateDirectory(h.ctx, versionDir); err != nil {
			return h.updateDatasetVersionStatus(dvCopy, dv, err)
		}
		dvCopy.Status.RootPath = versionDir

		// copy from other dataset version
		if err = h.copyFrom(b, versionDir, dv.Spec.CopyFrom); err != nil {
			return h.updateDatasetVersionStatus(dvCopy, dv, fmt.Errorf("copy failed: %w", err))
		}
	}

	// add version to dataset status
	if _, exist := versionExists(dataset.Status.Versions, dv.Spec.Version); !exist {
		datasetCopy := dataset.DeepCopy()
		datasetCopy.Status.Versions = append(datasetCopy.Status.Versions,
			mlv1.Version{Version: dv.Spec.Version, ObjectName: dv.Name})

		if _, err := h.updateDatasetStatus(datasetCopy, dataset, nil); err != nil {
			return h.updateDatasetVersionStatus(dvCopy, dv, fmt.Errorf("add version %s to dataset %s/%s failed: %w",
				dv.Spec.Version, dv.Namespace, dv.Spec.Dataset, err))
		}
	}

	// Handle publish functionality
	if dv.Spec.Publish && mlv1.Ready.IsTrue(dv) {
		if err := h.handlePublish(dvCopy); err != nil {
			return h.updateDatasetVersionStatus(dvCopy, dv, fmt.Errorf("publish failed: %w", err))
		}
	} else if !dv.Spec.Publish {
		// Cancel snapshot if publish is set to false
		if err := h.handleCancelPublish(dvCopy); err != nil {
			return h.updateDatasetVersionStatus(dvCopy, dv, fmt.Errorf("cancel publish failed: %w", err))
		}
	}

	return h.updateDatasetVersionStatus(dvCopy, dv, nil)
}

func (h *handler) OnRemoveDatasetVersion(_ string, dv *mlv1.DatasetVersion) (*mlv1.DatasetVersion, error) {
	if dv == nil || dv.Status.RootPath == "" || dv.DeletionTimestamp == nil {
		return nil, nil
	}

	logrus.Infof("delete dataset version %s(%s) of %s/%s", dv.Spec.Version, dv.Name, dv.Namespace, dv.Spec.Dataset)

	// delete directory of dataset version
	b, err := h.rm.NewBackendFromRegistry(h.ctx, dv.Status.Registry)
	if err != nil {
		return nil, fmt.Errorf(registry.ErrCreateBackendClient, err)
	}
	if err = b.Delete(h.ctx, dv.Status.RootPath); err != nil {
		return nil, fmt.Errorf("delete files of dataset version %s/%s/%s failed: %w",
			dv.Namespace, dv.Spec.Dataset, dv.Name, err)
	}

	// remove version from dataset status
	dataset, err := h.datasetCache.Get(dv.Namespace, dv.Spec.Dataset)
	if err != nil {
		if errors.IsNotFound(err) {
			return dv, nil
		}
		return nil, fmt.Errorf("get dataset %s/%s failed: %w", dv.Namespace, dv.Spec.Dataset, err)
	}
	if index, exist := versionExists(dataset.Status.Versions, dv.Spec.Version); exist {
		datasetCopy := dataset.DeepCopy()
		datasetCopy.Status.Versions = append(datasetCopy.Status.Versions[:index], datasetCopy.Status.Versions[index+1:]...)
		if _, err := h.updateDatasetStatus(datasetCopy, dataset, nil); err != nil {
			return nil, fmt.Errorf("remove version %s from dataset %s/%s failed: %w",
				dv.Spec.Version, dataset.Namespace, dataset.Name, err)
		}
	}

	return dv, nil
}

func (h *handler) copyFrom(b backend.Backend, dst string, copyFrom *mlv1.CopyFrom) error {
	if copyFrom == nil {
		return nil
	}

	logrus.Debugf("copy from %s/%s/%s", copyFrom.Namespace, copyFrom.Dataset, copyFrom.Version)

	// check if the source dataset version is exist
	dataset, err := h.datasetCache.Get(copyFrom.Namespace, copyFrom.Dataset)
	if err != nil {
		return fmt.Errorf("get dataset %s/%s failed: %w", copyFrom.Namespace, copyFrom.Dataset, err)
	}
	if !mlv1.Ready.IsTrue(dataset) {
		return fmt.Errorf("dataset %s/%s is not ready", copyFrom.Namespace, copyFrom.Dataset)
	}
	if _, exist := versionExists(dataset.Status.Versions, copyFrom.Version); !exist {
		return fmt.Errorf("version %s of dataset %s/%s not found", copyFrom.Version, copyFrom.Namespace, copyFrom.Dataset)
	}
	src := path.Join(dataset.Status.RootPath, copyFrom.Version)

	if err := b.Copy(h.ctx, src, dst); err != nil {
		return fmt.Errorf("copy from %s/%s/%s failed: %w", copyFrom.Namespace, copyFrom.Dataset, copyFrom.Version, err)
	}

	return nil
}

func (h *handler) updateDatasetVersionStatus(dvCopy, dv *mlv1.DatasetVersion, err error) (*mlv1.DatasetVersion, error) {
	if err == nil {
		mlv1.Ready.True(dvCopy)
		mlv1.Ready.Message(dvCopy, "")
	} else {
		mlv1.Ready.False(dvCopy)
		mlv1.Ready.Message(dvCopy, err.Error())
	}

	// don't update when no change happens
	if reflect.DeepEqual(dvCopy.Status, dv.Status) {
		return dvCopy, err
	}
	updatedDatasetVersion, updateErr := h.datasetVersionClient.UpdateStatus(dvCopy)
	if updateErr != nil {
		return nil, fmt.Errorf("update dataset version status failed: %w", updateErr)
	}
	return updatedDatasetVersion, err
}

func (h *handler) handlePublish(dv *mlv1.DatasetVersion) error {
	logrus.Infof("Starting publish process for dataset version %s/%s", dv.Namespace, dv.Name)

	// Create snapshotting spec
	spec := &snapshotting.Spec{
		Namespace: dv.Namespace,
		Name:      dv.Name,
		Labels: map[string]string{
			constant.LabelDatasetName:    dv.Spec.Dataset,
			constant.LabelDatasetVersion: dv.Spec.Version,
			constant.LabelResourceType:   "dataset-version",
		},
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: dv.APIVersion,
				Kind:       dv.Kind,
				Name:       dv.Name,
				UID:        dv.UID,
				Controller: ptr.To(true),
			},
		},
		PVCSpec: snapshotting.PVCSpec{
			AccessModes:               []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			RestoreFromLatestSnapshot: false, // For datasets, we don't restore from snapshots
		},
		JobSpec: snapshotting.JobSpec{
			BackoffLimit:            ptr.To(int32(1)),
			TTLSecondsAfterFinished: ptr.To(int32(86400)), // 24 hours
			Image:                   settings.ModelDownloaderImage.Get(),
			Args: []string{
				fmt.Sprintf("--name=%s/%s", dv.Namespace, dv.Name),
				fmt.Sprintf("--output-dir=%s", volumeMountPath),
				fmt.Sprintf("--type=%s", mlv1.DatasetVersionResourceName),
				"--debug=true",
			},
		},
	}

	// Call snapshotting manager
	return h.snapshotManager.DoSnapshot(h.ctx, spec)
}

func (h *handler) handleCancelPublish(dv *mlv1.DatasetVersion) error {
	// Build snapshotting spec for cancellation
	spec := &snapshotting.Spec{
		Namespace: dv.Namespace,
		Name:      dv.Name,
	}

	return h.snapshotManager.CancelSnapshot(h.ctx, spec)
}

func versionExists(versions []mlv1.Version, version string) (int, bool) {
	for i, v := range versions {
		if v.Version == version {
			return i, true
		}
	}
	return -1, false
}
