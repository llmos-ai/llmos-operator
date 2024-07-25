package fakeclients

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/generic"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	typecorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type ServiceClient func(string) typecorev1.ServiceInterface

func (n ServiceClient) Create(service *v1.Service) (*v1.Service, error) {
	return n(service.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
}

func (n ServiceClient) Update(service *v1.Service) (*v1.Service, error) {
	return n(service.Namespace).Update(context.TODO(), service, metav1.UpdateOptions{})
}

func (n ServiceClient) UpdateStatus(service *v1.Service) (*v1.Service, error) {
	return n(service.Namespace).UpdateStatus(context.TODO(), service, metav1.UpdateOptions{})
}

func (n ServiceClient) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return n(namespace).Delete(context.TODO(), name, *options)
}

func (n ServiceClient) Get(namespace, name string, options metav1.GetOptions) (*v1.Service, error) {
	return n(namespace).Get(context.TODO(), name, options)
}

func (n ServiceClient) List(namespace string, opts metav1.ListOptions) (*v1.ServiceList, error) {
	return n(namespace).List(context.TODO(), opts)
}

func (n ServiceClient) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return n(namespace).Watch(context.TODO(), opts)
}

func (n ServiceClient) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (*v1.Service, error) {
	return n(namespace).Patch(context.TODO(), name, pt, data, metav1.PatchOptions{}, subresources...)
}

func (n ServiceClient) WithImpersonation(_ rest.ImpersonationConfig) (generic.ClientInterface[*v1.Service, *v1.ServiceList], error) {
	panic("implement me")
}

type ServiceCache func(string) typecorev1.ServiceInterface

func (p ServiceCache) Get(namespace string, name string) (*v1.Service, error) {
	return p(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (p ServiceCache) List(namespace string, selector labels.Selector) ([]*v1.Service, error) {
	pods, err := p(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	result := make([]*v1.Service, 0, len(pods.Items))
	for _, pod := range pods.Items {
		obj := pod
		result = append(result, &obj)
	}
	return result, nil
}

func (p ServiceCache) AddIndexer(_ string, _ generic.Indexer[*v1.Service]) { // #nosec G101
	//TODO implement me
	panic("implement me")
}

func (p ServiceCache) GetByIndex(_ string, _ string) ([]*v1.Service, error) {
	//TODO implement me
	panic("implement me")
}
