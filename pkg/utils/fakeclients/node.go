package fakeclients

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/generic"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	corev1type "k8s.io/client-go/kubernetes/typed/core/v1"
)

type NodeCache func() corev1type.NodeInterface

func (c NodeCache) Get(name string) (*v1.Node, error) {
	return c().Get(context.TODO(), name, metav1.GetOptions{})
}
func (c NodeCache) List(selector labels.Selector) ([]*v1.Node, error) {
	list, err := c().List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*v1.Node, 0, len(list.Items))
	for _, item := range list.Items {
		obj := item
		result = append(result, &obj)
	}
	return result, err
}

func (c NodeCache) AddIndexer(_ string, _ generic.Indexer[*v1.Node]) {
	panic("implement me")
}
func (c NodeCache) GetByIndex(_, key string) ([]*v1.Node, error) {
	panic("implement me")
}
