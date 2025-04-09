package registry

import (
	"context"
	"fmt"

	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry/backend"
	"github.com/llmos-ai/llmos-operator/pkg/registry/backend/s3"
)

const (
	defaultSecretNamespace = "llmos-system"

	accessKeyIDName     = "accessKeyID"
	accessKeySecretName = "accessKeySecret"
)

type Manager struct {
	registryCache ctlmlv1.RegistryCache
	secretCache   ctlcorev1.SecretCache
}

func NewManager(secretCache ctlcorev1.SecretCache, registryCache ctlmlv1.RegistryCache) *Manager {
	return &Manager{
		registryCache: registryCache,
		secretCache:   secretCache,
	}
}

func (r *Manager) NewBackend(ctx context.Context, registry *mlv1.Registry) (backend.Backend, error) {
	// Get the secret containing access credentials from llmos-system namespace
	id, secret, err := getAccessKey(r.secretCache, registry.Spec.S3Config.AccessCredentialSecretName)
	if err != nil {
		return nil, fmt.Errorf("get access key failed: %w", err)
	}

	return s3.NewMinioClient(ctx, registry.Spec.S3Config.Endpoint, id, secret,
		registry.Spec.S3Config.Bucket, registry.Spec.S3Config.UseSSL)
}

func getAccessKey(secretCache ctlcorev1.SecretCache, accessCredentialSecretName string) (string, string, error) {
	secret, err := secretCache.Get(defaultSecretNamespace, accessCredentialSecretName)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", "", fmt.Errorf("secret %s not found in llmos-system namespace", accessCredentialSecretName)
		}
		return "", "", fmt.Errorf("get secret failed: %w", err)
	}

	// Extract credentials from the secret
	accessKeyID, ok := secret.Data[accessKeyIDName]
	if !ok {
		return "", "", fmt.Errorf("secret %s does not contain %s key", accessCredentialSecretName, accessKeyIDName)
	}

	accessKeySecret, ok := secret.Data[accessKeySecretName]
	if !ok {
		return "", "", fmt.Errorf("secret %s does not contain %s key", accessCredentialSecretName, accessKeySecretName)
	}

	return string(accessKeyID), string(accessKeySecret), nil
}

func (r *Manager) NewBackendFromRegistry(ctx context.Context, registryName string) (backend.Backend, error) {
	registry, err := r.registryCache.Get(registryName)
	if err != nil {
		return nil, fmt.Errorf("get registry %s failed: %w", registryName, err)
	}

	return r.NewBackend(ctx, registry)
}
