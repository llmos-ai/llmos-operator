package fakeclients

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	llmosv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	ctlllmosv1 "github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/typed/management.llmos.ai/v1"
)

type UpgradeClient func(string) ctlllmosv1.UpgradeInterface

func (n UpgradeClient) Create(upgrade *llmosv1.Upgrade) (*llmosv1.Upgrade, error) {
	return n(upgrade.Namespace).Create(context.TODO(), upgrade, metav1.CreateOptions{})
}

func (n UpgradeClient) Update(upgrade *llmosv1.Upgrade) (*llmosv1.Upgrade, error) {
	return n(upgrade.Namespace).Update(context.TODO(), upgrade, metav1.UpdateOptions{})
}

func (n UpgradeClient) UpdateStatus(upgrade *llmosv1.Upgrade) (*llmosv1.Upgrade, error) {
	return n(upgrade.Namespace).UpdateStatus(context.TODO(), upgrade, metav1.UpdateOptions{})
}

func (n UpgradeClient) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return n(namespace).Delete(context.TODO(), name, *options)
}

func (n UpgradeClient) Get(namespace, name string, options metav1.GetOptions) (*llmosv1.Upgrade, error) {
	return n(namespace).Get(context.TODO(), name, options)
}

func (n UpgradeClient) List(namespace string, opts metav1.ListOptions) (*llmosv1.UpgradeList, error) {
	return n(namespace).List(context.TODO(), opts)
}

func (n UpgradeClient) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return n(namespace).Watch(context.TODO(), opts)
}

func (n UpgradeClient) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (*llmosv1.Upgrade, error) {
	return n(namespace).Patch(context.TODO(), name, pt, data, metav1.PatchOptions{}, subresources...)
}

func (n UpgradeClient) WithImpersonation(_ rest.ImpersonationConfig) (generic.ClientInterface[*llmosv1.Upgrade, *llmosv1.UpgradeList], error) {
	panic("implement me")
}

type UpgradeCache func(string) ctlllmosv1.UpgradeInterface

func (u UpgradeCache) Get(namespace string, name string) (*llmosv1.Upgrade, error) {
	return u(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (u UpgradeCache) List(namespace string, selector labels.Selector) ([]*llmosv1.Upgrade, error) {
	pods, err := u(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	result := make([]*llmosv1.Upgrade, 0, len(pods.Items))
	for _, pod := range pods.Items {
		obj := pod
		result = append(result, &obj)
	}
	return result, nil
}

func (u UpgradeCache) AddIndexer(_ string, _ generic.Indexer[*llmosv1.Upgrade]) {
	//TODO implement me
	panic("implement me")
}

func (u UpgradeCache) GetByIndex(_ string, _ string) ([]*llmosv1.Upgrade, error) {
	//TODO implement me
	panic("implement me")
}
