package upgrade

import (
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
)

const (
	portName = "http"
)

func (h *upgradeHandler) reconcileUpgradeSystemChartsRepo(upgrade *mgmtv1.Upgrade) (*mgmtv1.Upgrade, error) {
	newVersion := upgrade.Spec.Version
	repo := constructUpgradeRepoDeployment(upgrade)
	foundRepo, err := h.deploymentCache.Get(repo.Namespace, repo.Name)
	if err != nil && apierrors.IsNotFound(err) {
		logrus.Debugf("Creating system charts repo for upgrade %s", upgrade.Name)
		if _, err = h.deploymentClient.Create(repo); err != nil {
			return h.updateErrorCond(upgrade, mgmtv1.UpgradeChartsRepoReady, err)
		}

		return h.updateUpgradingCond(upgrade, mgmtv1.UpgradeChartsRepoReady,
			fmt.Sprintf(msgWaitingForRepo, newVersion))
	}

	// create service for upgrade repo
	if err = h.reconcileRepoSvc(upgrade); err != nil {
		logrus.Debugf("Failed to reconcile upgrade service for upgrade %s: %v", upgrade.Name, err)
		return h.updateErrorCond(upgrade, mgmtv1.UpgradeChartsRepoReady, err)
	}

	repoUpdate := foundRepo.DeepCopy()
	latestRepoImage := formatRepoImage(upgrade.Spec.Registry, systemChartsImageName, newVersion)
	repoUpdate.Spec.Template.Spec.Containers[0].Image = latestRepoImage
	if !reflect.DeepEqual(repoUpdate.Spec, foundRepo.Spec) {
		logrus.Infof("Updating upgrade repo image to  %s", latestRepoImage)
		_, err = h.deploymentClient.Update(repoUpdate)
		if err != nil {
			return upgrade, err
		}

		return h.updateUpgradingCond(upgrade, mgmtv1.UpgradeChartsRepoReady,
			fmt.Sprintf("Upgrading repo version to %s", newVersion))
	}

	if DeploymentIsReady(repoUpdate) {
		return h.updateReadyCond(upgrade, mgmtv1.UpgradeChartsRepoReady,
			fmt.Sprintf("Chart repo is ready for upgrade %s", newVersion))
	}

	return upgrade, nil
}

func (h *upgradeHandler) reconcileRepoSvc(upgrade *mgmtv1.Upgrade) error {
	svc := constructSystemChartsRepoService(upgrade)
	_, err := h.svcCache.Get(svc.Namespace, svc.Name)
	if err != nil && apierror.IsNotFound(err) {
		if _, err = h.svcClient.Create(svc); err != nil {
			logrus.Errorf("Failed to create upgrade repo service for upgrade %s: %v", upgrade.Name, err)
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func constructUpgradeRepoDeployment(upgrade *mgmtv1.Upgrade) *v1.Deployment {
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      upgradeRepoName,
			Namespace: constant.SystemNamespaceName,
			Labels: map[string]string{
				llmosUpgradeNameLabel:      upgrade.Name,
				llmosUpgradeComponentLabel: upgradeRepoName,
				llmosVersionLabel:          upgrade.Spec.Version,
			},
		},
		Spec: v1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					llmosUpgradeComponentLabel: upgradeRepoName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						llmosUpgradeComponentLabel: upgradeRepoName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  upgradeRepoName,
							Image: formatRepoImage(upgrade.Spec.Registry, systemChartsImageName, upgrade.Spec.Version),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
									Name:          portName,
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
					Tolerations: getDefaultTolerations(),
				},
			},
		},
	}
}

func constructSystemChartsRepoService(upgrade *mgmtv1.Upgrade) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      upgradeRepoName,
			Namespace: constant.SystemNamespaceName,
			Labels: map[string]string{
				llmosUpgradeNameLabel:      upgrade.Name,
				llmosUpgradeComponentLabel: upgradeRepoName,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: portName,
					},
				},
			},
			Selector: map[string]string{
				llmosUpgradeComponentLabel: upgradeRepoName,
			},
		},
	}
}
