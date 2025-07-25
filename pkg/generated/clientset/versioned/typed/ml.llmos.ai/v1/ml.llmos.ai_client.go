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

package v1

import (
	http "net/http"

	mlllmosaiv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	scheme "github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/scheme"
	rest "k8s.io/client-go/rest"
)

type MlV1Interface interface {
	RESTClient() rest.Interface
	DatasetsGetter
	DatasetVersionsGetter
	LocalModelsGetter
	LocalModelVersionsGetter
	ModelsGetter
	ModelServicesGetter
	NotebooksGetter
	RegistriesGetter
}

// MlV1Client is used to interact with features provided by the ml.llmos.ai group.
type MlV1Client struct {
	restClient rest.Interface
}

func (c *MlV1Client) Datasets(namespace string) DatasetInterface {
	return newDatasets(c, namespace)
}

func (c *MlV1Client) DatasetVersions(namespace string) DatasetVersionInterface {
	return newDatasetVersions(c, namespace)
}

func (c *MlV1Client) LocalModels(namespace string) LocalModelInterface {
	return newLocalModels(c, namespace)
}

func (c *MlV1Client) LocalModelVersions(namespace string) LocalModelVersionInterface {
	return newLocalModelVersions(c, namespace)
}

func (c *MlV1Client) Models(namespace string) ModelInterface {
	return newModels(c, namespace)
}

func (c *MlV1Client) ModelServices(namespace string) ModelServiceInterface {
	return newModelServices(c, namespace)
}

func (c *MlV1Client) Notebooks(namespace string) NotebookInterface {
	return newNotebooks(c, namespace)
}

func (c *MlV1Client) Registries() RegistryInterface {
	return newRegistries(c)
}

// NewForConfig creates a new MlV1Client for the given config.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*MlV1Client, error) {
	config := *c
	setConfigDefaults(&config)
	httpClient, err := rest.HTTPClientFor(&config)
	if err != nil {
		return nil, err
	}
	return NewForConfigAndClient(&config, httpClient)
}

// NewForConfigAndClient creates a new MlV1Client for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
func NewForConfigAndClient(c *rest.Config, h *http.Client) (*MlV1Client, error) {
	config := *c
	setConfigDefaults(&config)
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &MlV1Client{client}, nil
}

// NewForConfigOrDie creates a new MlV1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *MlV1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new MlV1Client for the given RESTClient.
func New(c rest.Interface) *MlV1Client {
	return &MlV1Client{c}
}

func setConfigDefaults(config *rest.Config) {
	gv := mlllmosaiv1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = rest.CodecFactoryForGeneratedClient(scheme.Scheme, scheme.Codecs).WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *MlV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
