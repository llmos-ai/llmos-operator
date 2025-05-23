package downloader

import (
	"context"
	"fmt"
	"strings"

	ctlcore "github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apimodel "github.com/llmos-ai/llmos-operator/pkg/api/model"
	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlml "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	pkgreg "github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/server"
)

const (
	defaultThreadness = 3

	huggingfaceRegistry = "huggingface"
	modelScopeRegistry  = "modelscope"
)

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

func (c *client) Download(ctx context.Context, registry, modelName, outputDir string, threadness int) error {
	if registry == huggingfaceRegistry || registry == modelScopeRegistry {
		return nil
	}

	tmp := strings.Split(modelName, "/")
	if len(tmp) != 2 {
		return fmt.Errorf("invalid model name: %s", modelName)
	}
	namespace, name := tmp[0], tmp[1]

	reg, rootPath, err := apimodel.GetModelRegistryAndRootPath(c.getModel, namespace, name)
	if err != nil {
		return fmt.Errorf("failed to get registry and root path of %s/%s: %w", namespace, name, err)
	}

	b, err := pkgreg.NewManager(c.getSecret, c.getRegistry).NewBackendFromRegistry(ctx, reg)
	if err != nil {
		return fmt.Errorf("failed to create backend: %w", err)
	}

	if threadness <= 0 {
		threadness = defaultThreadness
	}
	return b.IncrementalDownload(ctx, rootPath, outputDir, threadness)
}

func (c *client) getModel(namespace, name string) (*mlv1.Model, error) {
	return c.LLMInterface.Model().Get(namespace, name, metav1.GetOptions{})
}

func (c *client) getRegistry(name string) (*mlv1.Registry, error) {
	return c.LLMInterface.Registry().Get(name, metav1.GetOptions{})
}

func (c *client) getSecret(namespace, name string) (*corev1.Secret, error) {
	return c.CoreInterface.Secret().Get(namespace, name, metav1.GetOptions{})
}
