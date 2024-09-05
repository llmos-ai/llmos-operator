package fakeclients

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/typed/management.llmos.ai/v1"
)

type GlobalRole func() ctlmgmtv1.GlobalRoleInterface

func (g GlobalRole) Create(gr *mgmtv1.GlobalRole) (*mgmtv1.GlobalRole, error) {
	return g().Create(context.TODO(), gr, metav1.CreateOptions{})
}

func (g GlobalRole) Update(gr *mgmtv1.GlobalRole) (*mgmtv1.GlobalRole, error) {
	return g().Update(context.TODO(), gr, metav1.UpdateOptions{})
}

func (g GlobalRole) UpdateStatus(gr *mgmtv1.GlobalRole) (*mgmtv1.GlobalRole, error) {
	return g().UpdateStatus(context.TODO(), gr, metav1.UpdateOptions{})
}

func (g GlobalRole) Delete(name string, opts *metav1.DeleteOptions) error {
	return g().Delete(context.TODO(), name, *opts)
}
func (g GlobalRole) Get(name string, options metav1.GetOptions) (*mgmtv1.GlobalRole, error) {
	return g().Get(context.TODO(), name, options)
}

func (g GlobalRole) List(opts metav1.ListOptions) (*mgmtv1.GlobalRoleList, error) {
	return g().List(context.TODO(), opts)
}

func (g GlobalRole) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return g().Watch(context.TODO(), opts)
}

func (g GlobalRole) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *mgmtv1.GlobalRole, err error) {
	return g().Patch(context.TODO(), name, pt, data, metav1.PatchOptions{}, subresources...)
}

func (g GlobalRole) WithImpersonation(_ rest.ImpersonationConfig) (generic.NonNamespacedClientInterface[*mgmtv1.GlobalRole, *mgmtv1.GlobalRoleList], error) {
	panic("implement me")
}
