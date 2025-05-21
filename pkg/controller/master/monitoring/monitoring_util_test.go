package monitoring

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// nolint:staticcheck
func TestConstructEtcdEndpointsSubset(t *testing.T) {
	tests := []struct {
		name     string
		nodes    []*corev1.Node
		expected []corev1.EndpointSubset
	}{
		{
			name:  "Empty nodes list",
			nodes: []*corev1.Node{},
			expected: []corev1.EndpointSubset{
				{
					Ports: []corev1.EndpointPort{
						{
							Name:     "http-metrics",
							Port:     2381,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			},
		},
		{
			name: "Nodes with internal IP",
			nodes: []*corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						UID:  "node1-uid",
					},
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{
							{Type: corev1.NodeInternalIP, Address: "192.168.1.1"},
						},
					},
				},
			},
			expected: []corev1.EndpointSubset{
				{
					Ports: []corev1.EndpointPort{
						{
							Name:     "http-metrics",
							Port:     2381,
							Protocol: corev1.ProtocolTCP,
						},
					},
					Addresses: []corev1.EndpointAddress{
						{
							IP: "192.168.1.1",
							TargetRef: &corev1.ObjectReference{
								Kind: "Node",
								Name: "node1",
								UID:  "node1-uid",
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := constructEtcdEndpointsSubset(tt.nodes)
			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("constructEtcdEndpointsSubset() = %v, want %v", actual, tt.expected)
			}
		})
	}
}
