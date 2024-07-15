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

package v1

import (
	"context"
	"time"

	v1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	scheme "github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// UpgradesGetter has a method to return a UpgradeInterface.
// A group's client should implement this interface.
type UpgradesGetter interface {
	Upgrades(namespace string) UpgradeInterface
}

// UpgradeInterface has methods to work with Upgrade resources.
type UpgradeInterface interface {
	Create(ctx context.Context, upgrade *v1.Upgrade, opts metav1.CreateOptions) (*v1.Upgrade, error)
	Update(ctx context.Context, upgrade *v1.Upgrade, opts metav1.UpdateOptions) (*v1.Upgrade, error)
	UpdateStatus(ctx context.Context, upgrade *v1.Upgrade, opts metav1.UpdateOptions) (*v1.Upgrade, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Upgrade, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.UpgradeList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Upgrade, err error)
	UpgradeExpansion
}

// upgrades implements UpgradeInterface
type upgrades struct {
	client rest.Interface
	ns     string
}

// newUpgrades returns a Upgrades
func newUpgrades(c *ManagementV1Client, namespace string) *upgrades {
	return &upgrades{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the upgrade, and returns the corresponding upgrade object, and an error if there is any.
func (c *upgrades) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Upgrade, err error) {
	result = &v1.Upgrade{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("upgrades").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Upgrades that match those selectors.
func (c *upgrades) List(ctx context.Context, opts metav1.ListOptions) (result *v1.UpgradeList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.UpgradeList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("upgrades").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested upgrades.
func (c *upgrades) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("upgrades").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a upgrade and creates it.  Returns the server's representation of the upgrade, and an error, if there is any.
func (c *upgrades) Create(ctx context.Context, upgrade *v1.Upgrade, opts metav1.CreateOptions) (result *v1.Upgrade, err error) {
	result = &v1.Upgrade{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("upgrades").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(upgrade).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a upgrade and updates it. Returns the server's representation of the upgrade, and an error, if there is any.
func (c *upgrades) Update(ctx context.Context, upgrade *v1.Upgrade, opts metav1.UpdateOptions) (result *v1.Upgrade, err error) {
	result = &v1.Upgrade{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("upgrades").
		Name(upgrade.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(upgrade).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *upgrades) UpdateStatus(ctx context.Context, upgrade *v1.Upgrade, opts metav1.UpdateOptions) (result *v1.Upgrade, err error) {
	result = &v1.Upgrade{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("upgrades").
		Name(upgrade.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(upgrade).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the upgrade and deletes it. Returns an error if one occurs.
func (c *upgrades) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("upgrades").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *upgrades) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("upgrades").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched upgrade.
func (c *upgrades) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Upgrade, err error) {
	result = &v1.Upgrade{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("upgrades").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
