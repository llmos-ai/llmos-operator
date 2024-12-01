package monitoring

import (
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/setting"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
)

func isManagementNode(node *corev1.Node) bool {
	if node.Labels[constant.KubeEtcdNodeLabelKey] == constant.TrueStr ||
		node.Labels[constant.KubeMasterNodeLabelKey] == constant.TrueStr ||
		node.Labels[constant.KubeControlPlaneNodeLabelKey] == constant.TrueStr {
		return true
	}
	return false
}

func isMonitoringEnabled() bool {
	cfgs := settings.ManagedAddonConfigs.Get()
	if cfgs == "" {
		return false
	}
	addonConfigs, err := setting.DecodeManagedAddonConfigs(cfgs)
	if err != nil {
		logrus.Errorf("failed to decode managed addon configs: %v", err)
		return false
	}

	return addonConfigs.LLMOSMonitoring.Enabled
}

// constructEtcdEndpointsSubset constructs the endpoint subset for the etcd monitoring service
func constructEtcdEndpointsSubset(nodes []*corev1.Node) []corev1.EndpointSubset {
	var endpointSubset = []corev1.EndpointSubset{
		{
			Ports: []corev1.EndpointPort{
				{
					Name:     "http-metrics",
					Port:     2381,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
	for _, node := range nodes {
		// Find node internal IP address
		for _, address := range node.Status.Addresses {
			if address.Type == corev1.NodeInternalIP {
				endpointSubset[0].Addresses = append(endpointSubset[0].Addresses, corev1.EndpointAddress{
					IP: address.Address,
					TargetRef: &corev1.ObjectReference{
						Kind: "Node",
						Name: node.Name,
						UID:  node.UID,
					},
				})
			}
		}
	}

	return endpointSubset
}
