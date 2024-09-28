package node

import (
	"context"

	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	nodeOnChangeName = "node.OnChange"
)

type handler struct {
	nodeClient ctlcorev1.NodeClient
	nodeCache  ctlcorev1.NodeCache
}

func Register(_ context.Context, mgmt *config.Management, _ config.Options) error {
	nodes := mgmt.CoreFactory.Core().V1().Node()
	h := handler{
		nodeClient: nodes,
		nodeCache:  nodes.Cache(),
	}
	nodes.OnChange(mgmt.Ctx, nodeOnChangeName, h.OnChange)
	return nil
}

func (h *handler) OnChange(_ string, node *corev1.Node) (*corev1.Node, error) {
	if node == nil || node.DeletionTimestamp != nil {
		return node, nil
	}

	return h.updateNodeLabels(node)
}

func (h *handler) updateNodeLabels(node *corev1.Node) (*corev1.Node, error) {
	if node.Labels != nil && node.Labels[constant.KubeWorkerNodeLabelKey] == "true" {
		return node, nil
	}

	update := node.DeepCopy()
	update.Labels[constant.KubeWorkerNodeLabelKey] = "true"

	return h.nodeClient.Update(update)
}
