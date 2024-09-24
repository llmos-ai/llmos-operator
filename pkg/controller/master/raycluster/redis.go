package raycluster

import (
	"fmt"
	"reflect"

	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
)

func (h *handler) syncGCSRedisSecretToNamespace(rayCluster *rayv1.RayCluster) error {
	// get redis secret for GCS config
	// if the secret is not ready, reconcile it
	redisSecret, err := h.secretsCache.Get(constant.SystemNamespaceName, defaultRedisName)
	if err != nil {
		return fmt.Errorf("failed to get system default redisSecret: %v", err)
	}

	// we only need to create 1 synced redis secret in the related namespace
	nsSecret, err := h.secretsCache.Get(rayCluster.Namespace, GetNameSpacedGCSSecretName(rayCluster.Namespace))
	if err != nil {
		if errors.IsNotFound(err) {
			newSecret := GetSyncedSecret(redisSecret, rayCluster)
			if _, err := h.secrets.Create(newSecret); err != nil {
				return fmt.Errorf("failed to sync redis secret in ns %s: %v", rayCluster.Namespace, err)
			}

		}
		return err
	}

	// sync GCS redis secret to the cluster namespace
	if !reflect.DeepEqual(redisSecret.Data, nsSecret.Data) {
		secretCpy := nsSecret.DeepCopy()
		secretCpy.Data = map[string][]byte{
			constant.RedisSecretKeyName: redisSecret.Data[constant.RedisSecretKeyName],
		}

		if _, err = h.secrets.Update(secretCpy); err != nil {
			return fmt.Errorf("failed to update secret %s: %v", GetNameSpacedGCSSecretName(rayCluster.Namespace), err)
		}
	}
	return nil
}

func GetGCSRedisSVCDomain() string {
	return fmt.Sprintf("redis://%s-master.%s.svc.cluster.local:6379", defaultRedisName, constant.SystemNamespaceName)
}

func GetNameSpacedGCSSecretName(namespace string) string {
	return fmt.Sprintf("%s-gcs-redis", namespace)
}

func GetSyncedSecret(redisSecret *corev1.Secret, cluster *rayv1.RayCluster) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetNameSpacedGCSSecretName(cluster.Namespace),
			Namespace: cluster.Namespace,
		},
		Data: map[string][]byte{
			constant.RedisSecretKeyName: redisSecret.Data[constant.RedisSecretKeyName],
		},
	}
}

func GetHeadNodeRedisEnvConfig(namespace string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "RAY_REDIS_ADDRESS",
			Value: GetGCSRedisSVCDomain(),
		},
		{
			Name: "REDIS_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: GetNameSpacedGCSSecretName(namespace),
					},
					Key: constant.RedisSecretKeyName,
				},
			},
		},
	}
}
