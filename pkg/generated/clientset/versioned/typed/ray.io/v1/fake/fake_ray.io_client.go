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
	v1 "github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/typed/ray.io/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeRayV1 struct {
	*testing.Fake
}

func (c *FakeRayV1) RayClusters() v1.RayClusterInterface {
	return &FakeRayClusters{c}
}

func (c *FakeRayV1) RayJobs() v1.RayJobInterface {
	return &FakeRayJobs{c}
}

func (c *FakeRayV1) RayServices() v1.RayServiceInterface {
	return &FakeRayServices{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeRayV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
