package fakeclients

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/generic"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	typeappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/rest"
)

type StatefulSetClient func(string) typeappsv1.StatefulSetInterface

func (n StatefulSetClient) Create(ss *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	return n(ss.Namespace).Create(context.TODO(), ss, metav1.CreateOptions{})
}

func (n StatefulSetClient) Update(ss *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	return n(ss.Namespace).Update(context.TODO(), ss, metav1.UpdateOptions{})
}

func (n StatefulSetClient) UpdateStatus(ss *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	return n(ss.Namespace).UpdateStatus(context.TODO(), ss, metav1.UpdateOptions{})
}

func (n StatefulSetClient) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return n(namespace).Delete(context.TODO(), name, *options)
}

func (n StatefulSetClient) Get(namespace, name string, options metav1.GetOptions) (*appsv1.StatefulSet, error) {
	return n(namespace).Get(context.TODO(), name, options)
}

func (n StatefulSetClient) List(namespace string, opts metav1.ListOptions) (*appsv1.StatefulSetList, error) {
	return n(namespace).List(context.TODO(), opts)
}

func (n StatefulSetClient) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return n(namespace).Watch(context.TODO(), opts)
}

func (n StatefulSetClient) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (*appsv1.StatefulSet, error) {
	return n(namespace).Patch(context.TODO(), name, pt, data, metav1.PatchOptions{}, subresources...)
}

func (n StatefulSetClient) WithImpersonation(_ rest.ImpersonationConfig) (generic.ClientInterface[*appsv1.StatefulSet, *appsv1.StatefulSetList], error) {
	panic("implement me")
}

type StatefulSetCache func(string) typeappsv1.StatefulSetInterface

func (p StatefulSetCache) Get(namespace string, name string) (*appsv1.StatefulSet, error) {
	return p(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (p StatefulSetCache) List(namespace string, selector labels.Selector) ([]*appsv1.StatefulSet, error) {
	pods, err := p(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	result := make([]*appsv1.StatefulSet, 0, len(pods.Items))
	for _, pod := range pods.Items {
		obj := pod
		result = append(result, &obj)
	}
	return result, nil
}

func (p StatefulSetCache) AddIndexer(_ string, _ generic.Indexer[*appsv1.StatefulSet]) {
	//TODO implement me
	panic("implement me")
}

func (p StatefulSetCache) GetByIndex(_ string, _ string) ([]*appsv1.StatefulSet, error) {
	//TODO implement me
	panic("implement me")
}
