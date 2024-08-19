/*
Copyright 2024 llmos.ai.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by main. DO NOT EDIT.

package fake

import (
	"context"

	v1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeModelServices implements ModelServiceInterface
type FakeModelServices struct {
	Fake *FakeMlV1
	ns   string
}

var modelservicesResource = v1.SchemeGroupVersion.WithResource("modelservices")

var modelservicesKind = v1.SchemeGroupVersion.WithKind("ModelService")

// Get takes name of the modelService, and returns the corresponding modelService object, and an error if there is any.
func (c *FakeModelServices) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.ModelService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(modelservicesResource, c.ns, name), &v1.ModelService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ModelService), err
}

// List takes label and field selectors, and returns the list of ModelServices that match those selectors.
func (c *FakeModelServices) List(ctx context.Context, opts metav1.ListOptions) (result *v1.ModelServiceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(modelservicesResource, modelservicesKind, c.ns, opts), &v1.ModelServiceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.ModelServiceList{ListMeta: obj.(*v1.ModelServiceList).ListMeta}
	for _, item := range obj.(*v1.ModelServiceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested modelServices.
func (c *FakeModelServices) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(modelservicesResource, c.ns, opts))

}

// Create takes the representation of a modelService and creates it.  Returns the server's representation of the modelService, and an error, if there is any.
func (c *FakeModelServices) Create(ctx context.Context, modelService *v1.ModelService, opts metav1.CreateOptions) (result *v1.ModelService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(modelservicesResource, c.ns, modelService), &v1.ModelService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ModelService), err
}

// Update takes the representation of a modelService and updates it. Returns the server's representation of the modelService, and an error, if there is any.
func (c *FakeModelServices) Update(ctx context.Context, modelService *v1.ModelService, opts metav1.UpdateOptions) (result *v1.ModelService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(modelservicesResource, c.ns, modelService), &v1.ModelService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ModelService), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeModelServices) UpdateStatus(ctx context.Context, modelService *v1.ModelService, opts metav1.UpdateOptions) (*v1.ModelService, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(modelservicesResource, "status", c.ns, modelService), &v1.ModelService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ModelService), err
}

// Delete takes name of the modelService and deletes it. Returns an error if one occurs.
func (c *FakeModelServices) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(modelservicesResource, c.ns, name, opts), &v1.ModelService{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeModelServices) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(modelservicesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1.ModelServiceList{})
	return err
}

// Patch applies the patch and returns the patched modelService.
func (c *FakeModelServices) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ModelService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(modelservicesResource, c.ns, name, pt, data, subresources...), &v1.ModelService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ModelService), err
}
