package fakeclients

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/generic"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	appsv1type "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/rest"
)

type DeploymentClient func(string) appsv1type.DeploymentInterface

func (d DeploymentClient) Update(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	return d(deployment.Namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
}

func (d DeploymentClient) Get(namespace, name string, options metav1.GetOptions) (*appsv1.Deployment, error) {
	return d(namespace).Get(context.TODO(), name, options)
}

func (d DeploymentClient) Create(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	return d(deployment.Namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
}
func (d DeploymentClient) UpdateStatus(*appsv1.Deployment) (*appsv1.Deployment, error) {
	panic("implement me")
}

func (d DeploymentClient) Delete(_, _ string, _ *metav1.DeleteOptions) error {
	panic("implement me")
}

func (d DeploymentClient) List(_ string, _ metav1.ListOptions) (*appsv1.DeploymentList, error) {
	panic("implement me")
}

func (d DeploymentClient) Watch(_ string, _ metav1.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (d DeploymentClient) Patch(_, _ string, _ types.PatchType, _ []byte, _ ...string) (result *appsv1.Deployment, err error) {
	panic("implement me")
}
func (d DeploymentClient) WithImpersonation(_ rest.ImpersonationConfig) (generic.ClientInterface[*appsv1.Deployment, *appsv1.DeploymentList], error) {
	panic("implement me")
}

type DeploymentCache func(string) appsv1type.DeploymentInterface

func (d DeploymentCache) Get(namespace, name string) (*appsv1.Deployment, error) {
	return d(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (d DeploymentCache) List(namespace string, selector labels.Selector) ([]*appsv1.Deployment, error) {
	deployments, err := d(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}

	result := make([]*appsv1.Deployment, 0, len(deployments.Items))
	for _, item := range deployments.Items {
		obj := item
		result = append(result, &obj)
	}
	return result, nil
}

func (d DeploymentCache) AddIndexer(indexName string, indexer generic.Indexer[*appsv1.Deployment]) {
	panic("implement me")
}

func (d DeploymentCache) GetByIndex(indexName, key string) ([]*appsv1.Deployment, error) {
	panic("implement me")
}
