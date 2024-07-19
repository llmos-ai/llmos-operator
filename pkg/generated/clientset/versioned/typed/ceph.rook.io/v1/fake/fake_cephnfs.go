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

	v1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeCephNFSs implements CephNFSInterface
type FakeCephNFSs struct {
	Fake *FakeCephV1
	ns   string
}

var cephnfssResource = v1.SchemeGroupVersion.WithResource("cephnfss")

var cephnfssKind = v1.SchemeGroupVersion.WithKind("CephNFS")

// Get takes name of the cephNFS, and returns the corresponding cephNFS object, and an error if there is any.
func (c *FakeCephNFSs) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.CephNFS, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(cephnfssResource, c.ns, name), &v1.CephNFS{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.CephNFS), err
}

// List takes label and field selectors, and returns the list of CephNFSs that match those selectors.
func (c *FakeCephNFSs) List(ctx context.Context, opts metav1.ListOptions) (result *v1.CephNFSList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(cephnfssResource, cephnfssKind, c.ns, opts), &v1.CephNFSList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.CephNFSList{ListMeta: obj.(*v1.CephNFSList).ListMeta}
	for _, item := range obj.(*v1.CephNFSList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested cephNFSs.
func (c *FakeCephNFSs) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(cephnfssResource, c.ns, opts))

}

// Create takes the representation of a cephNFS and creates it.  Returns the server's representation of the cephNFS, and an error, if there is any.
func (c *FakeCephNFSs) Create(ctx context.Context, cephNFS *v1.CephNFS, opts metav1.CreateOptions) (result *v1.CephNFS, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(cephnfssResource, c.ns, cephNFS), &v1.CephNFS{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.CephNFS), err
}

// Update takes the representation of a cephNFS and updates it. Returns the server's representation of the cephNFS, and an error, if there is any.
func (c *FakeCephNFSs) Update(ctx context.Context, cephNFS *v1.CephNFS, opts metav1.UpdateOptions) (result *v1.CephNFS, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(cephnfssResource, c.ns, cephNFS), &v1.CephNFS{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.CephNFS), err
}

// Delete takes name of the cephNFS and deletes it. Returns an error if one occurs.
func (c *FakeCephNFSs) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(cephnfssResource, c.ns, name, opts), &v1.CephNFS{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeCephNFSs) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(cephnfssResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1.CephNFSList{})
	return err
}

// Patch applies the patch and returns the patched cephNFS.
func (c *FakeCephNFSs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.CephNFS, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(cephnfssResource, c.ns, name, pt, data, subresources...), &v1.CephNFS{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.CephNFS), err
}