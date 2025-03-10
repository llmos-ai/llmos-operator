/*
Copyright 2025 llmos.ai.

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

	v1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeTokens implements TokenInterface
type FakeTokens struct {
	Fake *FakeManagementV1
}

var tokensResource = v1.SchemeGroupVersion.WithResource("tokens")

var tokensKind = v1.SchemeGroupVersion.WithKind("Token")

// Get takes name of the token, and returns the corresponding token object, and an error if there is any.
func (c *FakeTokens) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Token, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(tokensResource, name), &v1.Token{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Token), err
}

// List takes label and field selectors, and returns the list of Tokens that match those selectors.
func (c *FakeTokens) List(ctx context.Context, opts metav1.ListOptions) (result *v1.TokenList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(tokensResource, tokensKind, opts), &v1.TokenList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.TokenList{ListMeta: obj.(*v1.TokenList).ListMeta}
	for _, item := range obj.(*v1.TokenList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested tokens.
func (c *FakeTokens) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(tokensResource, opts))
}

// Create takes the representation of a token and creates it.  Returns the server's representation of the token, and an error, if there is any.
func (c *FakeTokens) Create(ctx context.Context, token *v1.Token, opts metav1.CreateOptions) (result *v1.Token, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(tokensResource, token), &v1.Token{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Token), err
}

// Update takes the representation of a token and updates it. Returns the server's representation of the token, and an error, if there is any.
func (c *FakeTokens) Update(ctx context.Context, token *v1.Token, opts metav1.UpdateOptions) (result *v1.Token, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(tokensResource, token), &v1.Token{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Token), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeTokens) UpdateStatus(ctx context.Context, token *v1.Token, opts metav1.UpdateOptions) (*v1.Token, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(tokensResource, "status", token), &v1.Token{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Token), err
}

// Delete takes name of the token and deletes it. Returns an error if one occurs.
func (c *FakeTokens) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(tokensResource, name, opts), &v1.Token{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeTokens) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(tokensResource, listOpts)

	_, err := c.Fake.Invokes(action, &v1.TokenList{})
	return err
}

// Patch applies the patch and returns the patched token.
func (c *FakeTokens) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Token, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(tokensResource, name, pt, data, subresources...), &v1.Token{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Token), err
}
