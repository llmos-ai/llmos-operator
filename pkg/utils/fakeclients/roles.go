package fakeclients

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/generic"
	//v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	rbacv1type "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
)

type RoleClient func(string) rbacv1type.RoleInterface

func (p RoleClient) Create(pod *rbacv1.Role) (*rbacv1.Role, error) {
	return p(pod.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
}

func (p RoleClient) Update(pod *rbacv1.Role) (*rbacv1.Role, error) {
	return p(pod.Namespace).Update(context.TODO(), pod, metav1.UpdateOptions{})
}

func (p RoleClient) UpdateStatus(pod *rbacv1.Role) (*rbacv1.Role, error) {
	panic("implement me")
}

func (p RoleClient) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return p(namespace).Delete(context.TODO(), name, *options)
}

func (p RoleClient) Get(namespace, name string, options metav1.GetOptions) (*rbacv1.Role, error) {
	return p(namespace).Get(context.TODO(), name, options)
}

func (p RoleClient) List(namespace string, opts metav1.ListOptions) (*rbacv1.RoleList, error) {
	return p(namespace).List(context.TODO(), opts)
}

func (p RoleClient) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return p(namespace).Watch(context.TODO(), opts)
}

func (p RoleClient) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *rbacv1.Role, err error) {
	return p(namespace).Patch(context.TODO(), name, pt, data, metav1.PatchOptions{}, subresources...)
}

func (p RoleClient) WithImpersonation(_ rest.ImpersonationConfig) (generic.ClientInterface[*rbacv1.Role, *rbacv1.RoleList], error) {
	panic("implement me")
}

type RoleCache func(string) rbacv1type.RoleInterface

func (r RoleCache) Get(namespace string, name string) (*rbacv1.Role, error) {
	return r(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (r RoleCache) List(namespace string, selector labels.Selector) ([]*rbacv1.Role, error) {
	roles, err := r(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	result := make([]*rbacv1.Role, 0, len(roles.Items))
	for _, pod := range roles.Items {
		obj := pod
		result = append(result, &obj)
	}
	return result, nil
}

func (r RoleCache) AddIndexer(_ string, _ generic.Indexer[*rbacv1.Role]) {
	panic("implement me")
}

func (r RoleCache) GetByIndex(_ string, _ string) ([]*rbacv1.Role, error) {
	//TODO implement me
	panic("implement me")
}
