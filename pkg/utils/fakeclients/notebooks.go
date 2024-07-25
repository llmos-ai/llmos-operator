package fakeclients

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/typed/ml.llmos.ai/v1"
)

type NotebookClient func(string) ctlmlv1.NotebookInterface

func (n NotebookClient) Create(notebook *mlv1.Notebook) (*mlv1.Notebook, error) {
	return n(notebook.Namespace).Create(context.TODO(), notebook, metav1.CreateOptions{})
}

func (n NotebookClient) Update(notebook *mlv1.Notebook) (*mlv1.Notebook, error) {
	return n(notebook.Namespace).Update(context.TODO(), notebook, metav1.UpdateOptions{})
}

func (n NotebookClient) UpdateStatus(notebook *mlv1.Notebook) (*mlv1.Notebook, error) {
	return n(notebook.Namespace).UpdateStatus(context.TODO(), notebook, metav1.UpdateOptions{})
}

func (n NotebookClient) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	return n(namespace).Delete(context.TODO(), name, *options)
}

func (n NotebookClient) Get(namespace, name string, options metav1.GetOptions) (*mlv1.Notebook, error) {
	return n(namespace).Get(context.TODO(), name, options)
}

func (n NotebookClient) List(namespace string, opts metav1.ListOptions) (*mlv1.NotebookList, error) {
	return n(namespace).List(context.TODO(), opts)
}

func (n NotebookClient) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return n(namespace).Watch(context.TODO(), opts)
}

func (n NotebookClient) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (*mlv1.Notebook, error) {
	return n(namespace).Patch(context.TODO(), name, pt, data, metav1.PatchOptions{}, subresources...)
}

func (n NotebookClient) WithImpersonation(_ rest.ImpersonationConfig) (generic.ClientInterface[*mlv1.Notebook, *mlv1.NotebookList], error) {
	panic("implement me")
}
