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

type UpgradeClient func() ctlllmosv1.UpgradeInterface

func (n UpgradeClient) Create(upgrade *llmosv1.Upgrade) (*llmosv1.Upgrade, error) {
	return n().Create(context.TODO(), upgrade, metav1.CreateOptions{})
}

func (n UpgradeClient) Update(upgrade *llmosv1.Upgrade) (*llmosv1.Upgrade, error) {
	return n().Update(context.TODO(), upgrade, metav1.UpdateOptions{})
}

func (n UpgradeClient) UpdateStatus(upgrade *llmosv1.Upgrade) (*llmosv1.Upgrade, error) {
	return n().UpdateStatus(context.TODO(), upgrade, metav1.UpdateOptions{})
}

func (n UpgradeClient) Delete(name string, options *metav1.DeleteOptions) error {
	return n().Delete(context.TODO(), name, *options)
}

func (n UpgradeClient) Get(name string, options metav1.GetOptions) (*llmosv1.Upgrade, error) {
	return n().Get(context.TODO(), name, options)
}

func (n UpgradeClient) List(opts metav1.ListOptions) (*llmosv1.UpgradeList, error) {
	return n().List(context.TODO(), opts)
}

func (n UpgradeClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return n().Watch(context.TODO(), opts)
}

func (n UpgradeClient) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (*llmosv1.Upgrade, error) {
	return n().Patch(context.TODO(), name, pt, data, metav1.PatchOptions{}, subresources...)
}

func (n UpgradeClient) WithImpersonation(_ rest.ImpersonationConfig) (generic.NonNamespacedClientInterface[*llmosv1.Upgrade, *llmosv1.UpgradeList], error) {
	panic("implement me")
}

type UpgradeCache func() ctlllmosv1.UpgradeInterface

func (u UpgradeCache) Get(name string) (*llmosv1.Upgrade, error) {
	return u().Get(context.TODO(), name, metav1.GetOptions{})
}

func (u UpgradeCache) List(selector labels.Selector) ([]*llmosv1.Upgrade, error) {
	pods, err := u().List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
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
