package fakeclients

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	typedmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/typed/management.llmos.ai/v1"
)

type ManagedAddonClient func(string) typedmgmtv1.ManagedAddonInterface

func (m ManagedAddonClient) Create(addon *mgmtv1.ManagedAddon) (*mgmtv1.ManagedAddon, error) {
	return m(addon.Namespace).Create(context.TODO(), addon, metav1.CreateOptions{})
}

func (m ManagedAddonClient) Update(addon *mgmtv1.ManagedAddon) (*mgmtv1.ManagedAddon, error) {
	return m(addon.Namespace).Update(context.TODO(), addon, metav1.UpdateOptions{})
}

func (m ManagedAddonClient) UpdateStatus(addon *mgmtv1.ManagedAddon) (*mgmtv1.ManagedAddon, error) {
	return m(addon.Namespace).UpdateStatus(context.TODO(), addon, metav1.UpdateOptions{})
}

func (m ManagedAddonClient) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return m(namespace).Delete(context.TODO(), name, *options)
}

func (m ManagedAddonClient) Get(namespace, name string, options metav1.GetOptions) (*mgmtv1.ManagedAddon, error) {
	return m(namespace).Get(context.TODO(), name, options)
}

func (m ManagedAddonClient) List(namespace string, opts metav1.ListOptions) (*mgmtv1.ManagedAddonList, error) {
	return m(namespace).List(context.TODO(), opts)
}

func (m ManagedAddonClient) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return m(namespace).Watch(context.TODO(), opts)
}

func (m ManagedAddonClient) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *mgmtv1.ManagedAddon, err error) {
	//TODO implement me
	panic("implement me")
}

func (m ManagedAddonClient) WithImpersonation(impersonate rest.ImpersonationConfig) (generic.ClientInterface[*mgmtv1.ManagedAddon, *mgmtv1.ManagedAddonList], error) {
	//TODO implement me
	panic("implement me")
}

type ManagedAddonCache func(string) typedmgmtv1.ManagedAddonInterface

func (m ManagedAddonCache) Get(namespace, name string) (*mgmtv1.ManagedAddon, error) {
	return m(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (m ManagedAddonCache) List(namespace string, selector labels.Selector) ([]*mgmtv1.ManagedAddon, error) {
	list, err := m(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	result := make([]*mgmtv1.ManagedAddon, 0, len(list.Items))
	for _, item := range list.Items {
		obj := item
		result = append(result, &obj)
	}
	return result, nil
}

func (m ManagedAddonCache) AddIndexer(indexName string, indexer generic.Indexer[*mgmtv1.ManagedAddon]) {
	panic("implement me")
}

func (m ManagedAddonCache) GetByIndex(indexName, key string) ([]*mgmtv1.ManagedAddon, error) {
	panic("implement me")
}
