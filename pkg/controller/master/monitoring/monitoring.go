package monitoring

import (
	"context"
	"reflect"

	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	nodeOnChange                = "monitoring.onNodeChange"
	monitoringEtcdEndpointsName = "llmos-monitoring-kube-etcd"
)

type handler struct {
	nodeCache      ctlcorev1.NodeCache
	endpointClient ctlcorev1.EndpointsClient
	endpointsCache ctlcorev1.EndpointsCache
}

func Register(ctx context.Context, mgmt *config.Management, _ config.Options) error {
	node := mgmt.CoreFactory.Core().V1().Node()
	endpoints := mgmt.CoreFactory.Core().V1().Endpoints()
	h := &handler{
		nodeCache:      node.Cache(),
		endpointClient: endpoints,
		endpointsCache: endpoints.Cache(),
	}

	node.OnChange(ctx, nodeOnChange, h.onNodeChange)
	return nil
}

func (h *handler) onNodeChange(_ string, node *corev1.Node) (*corev1.Node, error) {
	if node == nil || node.DeletionTimestamp != nil {
		return nil, nil
	}

	if !isManagementNode(node) {
		return node, nil
	}

	if !isMonitoringEnabled() {
		logrus.Debugf("Monitoring is not enabled, skipping monitoring sync")
		return node, nil
	}

	if err := h.registerEtcdMetricsEndpoint(); err != nil {
		return node, err
	}

	return node, nil
}

func (h *handler) registerEtcdMetricsEndpoint() error {
	masterNodes, err := h.nodeCache.List(labels.SelectorFromSet(map[string]string{
		constant.KubeMasterNodeLabelKey: "true",
	}))
	if err != nil {
		return err
	}

	if len(masterNodes) <= 0 {
		logrus.Debugf("No master nodes found, skipping etcd metrics etcdMetricsEndpoint registration")
		return nil
	}

	etcdMetricsEndpoint, err := h.endpointsCache.Get(constant.KubeSystemNamespaceName, monitoringEtcdEndpointsName)
	if err != nil {
		return err
	}

	endpointSubsets := constructEtcdEndpointsSubset(masterNodes)
	if !reflect.DeepEqual(etcdMetricsEndpoint.Subsets, endpointSubsets) {
		logrus.Debugf("Updating etcdMetricsEndpoint subsets: %v", endpointSubsets)
		endpointCpy := etcdMetricsEndpoint.DeepCopy()
		endpointCpy.Subsets = endpointSubsets
		_, err = h.endpointClient.Update(endpointCpy)
		if err != nil {
			return err
		}
	}

	return nil
}
