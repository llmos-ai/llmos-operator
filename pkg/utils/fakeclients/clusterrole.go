package fakeclients

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/generic"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	rbacv1type "k8s.io/client-go/kubernetes/typed/rbac/v1"
)

type ClusterRole func() rbacv1type.ClusterRoleInterface

func (c ClusterRole) Create(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	return c().Create(context.TODO(), cr, metav1.CreateOptions{})
}

func (c ClusterRole) Update(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	return c().Update(context.TODO(), cr, metav1.UpdateOptions{})
}

func (c ClusterRole) UpdateStatus(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
	panic("implement me")
}

func (c ClusterRole) Delete(name string, opts *metav1.DeleteOptions) error {
	return c().Delete(context.TODO(), name, *opts)
}
func (c ClusterRole) Get(name string, options metav1.GetOptions) (*rbacv1.ClusterRole, error) {
	return c().Get(context.TODO(), name, options)
}

func (c ClusterRole) List(opts metav1.ListOptions) (*rbacv1.ClusterRoleList, error) {
	return c().List(context.TODO(), opts)
}

func (c ClusterRole) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c().Watch(context.TODO(), opts)
}

func (c ClusterRole) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *rbacv1.ClusterRole, err error) {
	return c().Patch(context.TODO(), name, pt, data, metav1.PatchOptions{}, subresources...)
}

func (c ClusterRole) WithImpersonation(_ rest.ImpersonationConfig) (generic.NonNamespacedClientInterface[*rbacv1.ClusterRole, *rbacv1.ClusterRoleList], error) {
	panic("implement me")
}

type ClusterRoleCache func() rbacv1type.ClusterRoleInterface

func (c ClusterRoleCache) Get(name string) (*rbacv1.ClusterRole, error) {
	return c().Get(context.TODO(), name, metav1.GetOptions{})
}

func (c ClusterRoleCache) List(selector labels.Selector) ([]*rbacv1.ClusterRole, error) {
	crs, err := c().List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}

	result := make([]*rbacv1.ClusterRole, len(crs.Items))
	for _, cr := range crs.Items {
		result = append(result, cr.DeepCopy())
	}
	return result, nil
}

func (c ClusterRoleCache) AddIndexer(_ string, _ generic.Indexer[*rbacv1.ClusterRole]) {
	panic("implement me")
}

func (c ClusterRoleCache) GetByIndex(_ string, _ string) ([]*rbacv1.ClusterRole, error) {
	panic("implement me")
}
