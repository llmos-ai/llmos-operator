package downloader

import (
	"context"
	"fmt"

	ctlcore "github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apidv "github.com/llmos-ai/llmos-operator/pkg/api/datasetversion"
	apimodel "github.com/llmos-ai/llmos-operator/pkg/api/model"
	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlml "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	pkgreg "github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/server"
)

const defaultThreadness = 3

type client struct {
	LLMInterface  ctlmlv1.Interface
	CoreInterface ctlcorev1.Interface
}

func newClient(kubeConfig string) (*client, error) {
	// Initialize Kubernetes client
	clientConfig, err := server.GetConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Get REST config for dynamic client
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get REST config: %w", err)
	}

	llm, err := ctlml.NewFactoryFromConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ml client: %w", err)
	}

	core, err := ctlcore.NewFactoryFromConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create core client: %w", err)
	}

	return &client{
		LLMInterface:  llm.Ml().V1(),
		CoreInterface: core.Core().V1(),
	}, nil
}

func (c *client) Download(ctx context.Context, resourceType, namespace, name, outputDir string, threadness int) error {
	registry, rootPath, err := c.getRegistryAndRootPath(resourceType, namespace, name)
	if err != nil {
		return fmt.Errorf("failed to get registry and root path of %s(%s/%s): %w", resourceType, namespace, name, err)
	}

	b, err := pkgreg.NewManager(c.getSecret, c.getRegistry).NewBackendFromRegistry(ctx, registry)
	if err != nil {
		return fmt.Errorf("failed to create backend: %w", err)
	}

	if threadness <= 0 {
		threadness = defaultThreadness
	}
	return b.IncrementalDownload(ctx, rootPath, outputDir, threadness)
}

func (c *client) getRegistryAndRootPath(resourceType, namespace, name string) (string, string, error) {
	switch resourceType {
	case mlv1.ModelResourceName:
		return apimodel.GetModelRegistryAndRootPath(c.getModel, namespace, name)
	case mlv1.DatasetVersionResourceName:
		return apidv.GetDatasetVersionRegistryAndRootPath(c.getDatasetVersion, namespace, name)
	default:
		return "", "", fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

func (c *client) getModel(namespace, name string) (*mlv1.Model, error) {
	return c.LLMInterface.Model().Get(namespace, name, metav1.GetOptions{})
}

func (c *client) getDatasetVersion(namespace, name string) (*mlv1.DatasetVersion, error) {
	return c.LLMInterface.DatasetVersion().Get(namespace, name, metav1.GetOptions{})
}

func (c *client) getRegistry(name string) (*mlv1.Registry, error) {
	return c.LLMInterface.Registry().Get(name, metav1.GetOptions{})
}

func (c *client) getSecret(namespace, name string) (*corev1.Secret, error) {
	return c.CoreInterface.Secret().Get(namespace, name, metav1.GetOptions{})
}
