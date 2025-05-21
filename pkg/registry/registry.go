package registry

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry/backend"
	"github.com/llmos-ai/llmos-operator/pkg/registry/backend/s3"
)

const (
	defaultSecretNamespace = "llmos-system"

	accessKeyIDName     = "accessKeyID"
	accessKeySecretName = "accessKeySecret"
)

type RegistryGetter func(name string) (*mlv1.Registry, error)
type SecretGetter func(namespace, name string) (*corev1.Secret, error)

type Manager struct {
	RegistryGetter
	SecretGetter
}

func NewManager(sg SecretGetter, rg RegistryGetter) *Manager {
	return &Manager{
		rg,
		sg,
	}
}

func (r *Manager) NewBackend(ctx context.Context, registry *mlv1.Registry) (backend.Backend, error) {
	// Get the secret containing access credentials from llmos-system namespace
	id, secret, err := getAccessKey(r.SecretGetter, registry.Spec.S3Config.AccessCredentialSecretName)
	if err != nil {
		return nil, fmt.Errorf("get access key failed: %w", err)
	}

	return s3.NewMinioClient(ctx, registry.Spec.S3Config.Endpoint, id, secret,
		registry.Spec.S3Config.Bucket, registry.Spec.S3Config.UseSSL)
}

func getAccessKey(sg SecretGetter, accessCredentialSecretName string) (string, string, error) {
	secret, err := sg(defaultSecretNamespace, accessCredentialSecretName)
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
	registry, err := r.RegistryGetter(registryName)
	if err != nil {
		return nil, fmt.Errorf("get registry %s failed: %w", registryName, err)
	}

	return r.NewBackend(ctx, registry)
}
