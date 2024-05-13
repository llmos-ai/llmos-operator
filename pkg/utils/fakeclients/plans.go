package fakeclients

import (
	"context"

	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	"github.com/rancher/wrangler/v2/pkg/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	ctlupgradev1 "github.com/llmos-ai/llmos-controller/pkg/generated/clientset/versioned/typed/upgrade.cattle.io/v1"
)

type PlanClient func(string) ctlupgradev1.PlanInterface

func (n PlanClient) Create(plan *upgradev1.Plan) (*upgradev1.Plan, error) {
	return n(plan.Namespace).Create(context.TODO(), plan, metav1.CreateOptions{})
}

func (n PlanClient) Update(plan *upgradev1.Plan) (*upgradev1.Plan, error) {
	return n(plan.Namespace).Update(context.TODO(), plan, metav1.UpdateOptions{})
}

func (n PlanClient) UpdateStatus(plan *upgradev1.Plan) (*upgradev1.Plan, error) {
	return n(plan.Namespace).UpdateStatus(context.TODO(), plan, metav1.UpdateOptions{})
}

func (n PlanClient) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return n(namespace).Delete(context.TODO(), name, *options)
}

func (n PlanClient) Get(namespace, name string, options metav1.GetOptions) (*upgradev1.Plan, error) {
	return n(namespace).Get(context.TODO(), name, options)
}

func (n PlanClient) List(namespace string, opts metav1.ListOptions) (*upgradev1.PlanList, error) {
	return n(namespace).List(context.TODO(), opts)
}

func (n PlanClient) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return n(namespace).Watch(context.TODO(), opts)
}

func (n PlanClient) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (*upgradev1.Plan, error) {
	return n(namespace).Patch(context.TODO(), name, pt, data, metav1.PatchOptions{}, subresources...)
}

func (n PlanClient) WithImpersonation(_ rest.ImpersonationConfig) (generic.ClientInterface[*upgradev1.Plan, *upgradev1.PlanList], error) {
	panic("implement me")
}

type PlanCache func(string) ctlupgradev1.PlanInterface

func (u PlanCache) Get(namespace string, name string) (*upgradev1.Plan, error) {
	return u(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (u PlanCache) List(namespace string, selector labels.Selector) ([]*upgradev1.Plan, error) {
	pods, err := u(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	result := make([]*upgradev1.Plan, 0, len(pods.Items))
	for _, pod := range pods.Items {
		obj := pod
		result = append(result, &obj)
	}
	return result, nil
}

func (u PlanCache) AddIndexer(_ string, _ generic.Indexer[*upgradev1.Plan]) {
	//TODO implement me
	panic("implement me")
}

func (u PlanCache) GetByIndex(_ string, _ string) ([]*upgradev1.Plan, error) {
	//TODO implement me
	panic("implement me")
}
