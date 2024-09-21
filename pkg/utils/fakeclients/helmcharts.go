package fakeclients

import (
	"context"

	helmv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	"github.com/rancher/wrangler/v3/pkg/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	typedhelmv1 "github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/typed/helm.cattle.io/v1"
)

type HelmChartClient func(string) typedhelmv1.HelmChartInterface

func (c HelmChartClient) Create(chart *helmv1.HelmChart) (*helmv1.HelmChart, error) {
	return c(chart.Namespace).Create(context.TODO(), chart, metav1.CreateOptions{})
}

func (c HelmChartClient) Update(plan *helmv1.HelmChart) (*helmv1.HelmChart, error) {
	return c(plan.Namespace).Update(context.TODO(), plan, metav1.UpdateOptions{})
}

func (c HelmChartClient) UpdateStatus(plan *helmv1.HelmChart) (*helmv1.HelmChart, error) {
	return c(plan.Namespace).UpdateStatus(context.TODO(), plan, metav1.UpdateOptions{})
}

func (c HelmChartClient) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return c(namespace).Delete(context.TODO(), name, *options)
}

func (c HelmChartClient) Get(namespace, name string, options metav1.GetOptions) (*helmv1.HelmChart, error) {
	return c(namespace).Get(context.TODO(), name, options)
}

func (c HelmChartClient) List(namespace string, opts metav1.ListOptions) (*helmv1.HelmChartList, error) {
	return c(namespace).List(context.TODO(), opts)
}

func (c HelmChartClient) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return c(namespace).Watch(context.TODO(), opts)
}

func (c HelmChartClient) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (*helmv1.HelmChart, error) {
	return c(namespace).Patch(context.TODO(), name, pt, data, metav1.PatchOptions{}, subresources...)
}

func (c HelmChartClient) WithImpersonation(_ rest.ImpersonationConfig) (generic.ClientInterface[*helmv1.HelmChart, *helmv1.HelmChartList], error) {
	panic("implement me")
}

type HelmChartCache func(string) typedhelmv1.HelmChartInterface

func (c HelmChartCache) Get(namespace string, name string) (*helmv1.HelmChart, error) {
	return c(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (c HelmChartCache) List(namespace string, selector labels.Selector) ([]*helmv1.HelmChart, error) {
	list, err := c(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	result := make([]*helmv1.HelmChart, 0, len(list.Items))
	for _, item := range list.Items {
		obj := item
		result = append(result, &obj)
	}
	return result, nil
}

func (c HelmChartCache) AddIndexer(_ string, _ generic.Indexer[*helmv1.HelmChart]) {
	panic("implement me")
}

func (c HelmChartCache) GetByIndex(_ string, _ string) ([]*helmv1.HelmChart, error) {
	panic("implement me")
}
