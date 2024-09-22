package storage

import (
	"fmt"

	rookv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
)

const (
	defaultMinimumNodesToScale = 3
)

func (h *Handler) onNodeChanged(_ string, node *corev1.Node) (*corev1.Node, error) {
	if node == nil || node.DeletionTimestamp != nil {
		return nil, nil
	}

	if err := h.autoScaleSystemCephClusterCount(); err != nil {
		return node, err
	}

	return nil, nil
}

// autoScaleSystemCephClusterCount will auto-scale system ceph cluster's mon and mgr count upon available nodes
func (h *Handler) autoScaleSystemCephClusterCount() error {
	cluster, err := h.clusterCache.Get(constant.CephSystemNamespaceName, constant.CephClusterName)
	if err != nil {
		return fmt.Errorf("failed to get system ceph cluster, error: %s", err.Error())
	}

	if cluster.Status.Phase != rookv1.ConditionReady {
		return fmt.Errorf("system ceph cluster is not ready, reconcile and process it later")
	}

	monCount := cluster.Spec.Mon.Count
	mgrCount := cluster.Spec.Mgr.Count
	if monCount >= 3 {
		logrus.Info("system ceph cluster already has minimum required mon and mgr count, " +
			"please scale it manually if needed")
		return nil
	}

	count, err := h.checkAvailableNodes()
	if err != nil {
		return fmt.Errorf("failed to check minimum available nodes, error: %s", err.Error())
	}

	if count > monCount {
		clusterCpy := cluster.DeepCopy()
		if monCount < 3 {
			clusterCpy.Spec.Mon.Count = 3
		}

		if mgrCount < 2 {
			clusterCpy.Spec.Mgr.Count = 2
		}

		if _, err = h.clusters.Update(clusterCpy); err != nil {
			return fmt.Errorf("failed to update system ceph cluster, error: %s", err.Error())
		}
	}

	return nil
}

func (h *Handler) checkAvailableNodes() (int, error) {
	nodeList, err := h.nodeCache.List(labels.Everything())
	if err != nil {
		return 0, fmt.Errorf("failed to list node, error: %s", err.Error())
	}

	if len(nodeList) < defaultMinimumNodesToScale {
		return 0, nil
	}

	count := 0
	for _, node := range nodeList {
		if node.Spec.Unschedulable {
			continue
		}

		if node.Status.Conditions == nil {
			continue
		}

		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				count++
			}
		}
	}

	return count, nil
}
