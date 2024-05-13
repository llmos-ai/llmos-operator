package fakeclients

import (
	"context"

	"github.com/rancher/wrangler/v2/pkg/generic"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	corev1type "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type PodClient func(string) corev1type.PodInterface

func (p PodClient) Create(pod *v1.Pod) (*v1.Pod, error) {
	return p(pod.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
}

func (p PodClient) Update(pod *v1.Pod) (*v1.Pod, error) {
	return p(pod.Namespace).Update(context.TODO(), pod, metav1.UpdateOptions{})
}

func (p PodClient) UpdateStatus(pod *v1.Pod) (*v1.Pod, error) {
	return p(pod.Namespace).UpdateStatus(context.TODO(), pod, metav1.UpdateOptions{})
}

func (p PodClient) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return p(namespace).Delete(context.TODO(), name, *options)
}

func (p PodClient) Get(namespace, name string, options metav1.GetOptions) (*v1.Pod, error) {
	return p(namespace).Get(context.TODO(), name, options)
}

func (p PodClient) List(namespace string, opts metav1.ListOptions) (*v1.PodList, error) {
	return p(namespace).List(context.TODO(), opts)
}

func (p PodClient) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return p(namespace).Watch(context.TODO(), opts)
}

func (p PodClient) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Pod, err error) {
	return p(namespace).Patch(context.TODO(), name, pt, data, metav1.PatchOptions{}, subresources...)
}

func (p PodClient) WithImpersonation(_ rest.ImpersonationConfig) (generic.ClientInterface[*v1.Pod, *v1.PodList], error) {
	panic("implement me")
}

type PodCache func(string) corev1type.PodInterface

func (u PodCache) Get(namespace string, name string) (*v1.Pod, error) {
	return u(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (u PodCache) List(namespace string, selector labels.Selector) ([]*v1.Pod, error) {
	pods, err := u(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	result := make([]*v1.Pod, 0, len(pods.Items))
	for _, pod := range pods.Items {
		obj := pod
		result = append(result, &obj)
	}
	return result, nil
}

func (u PodCache) AddIndexer(_ string, _ generic.Indexer[*v1.Pod]) {
	//TODO implement me
	panic("implement me")
}

func (u PodCache) GetByIndex(_ string, _ string) ([]*v1.Pod, error) {
	//TODO implement me
	panic("implement me")
}
