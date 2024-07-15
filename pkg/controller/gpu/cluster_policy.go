package gpu

import (
	"context"
	"fmt"
	"strings"

	nvidiav1 "github.com/NVIDIA/gpu-operator/api/v1"
	detector "github.com/rancher/kubernetes-provider-detector"
	"k8s.io/client-go/kubernetes"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlnvidiav1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/nvidia.com/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

var (
	RuntimeK3S  Runtime = "k3s"
	RuntimeRKE2 Runtime = "rke2"
)

type Runtime string

type handler struct {
	ctx       context.Context
	k8sClient kubernetes.Interface
	policies  ctlnvidiav1.ClusterPolicyClient
}

func Register(ctx context.Context, mgmt *config.Management) error {
	policies := mgmt.NvidiaFactory.Nvidia().V1().ClusterPolicy()
	h := handler{
		ctx:       ctx,
		k8sClient: mgmt.ClientSet,
		policies:  policies,
	}

	policies.OnChange(ctx, "ob-gpu-cluster-policy", h.OnChanged)
	return nil
}

func (h *handler) OnChanged(_ string, policy *nvidiav1.ClusterPolicy) (*nvidiav1.ClusterPolicy, error) {
	if policy == nil || policy.DeletionTimestamp != nil {
		return policy, nil
	}

	_, err := h.configToolkitContainerd(policy)
	if err != nil {
		return policy, err
	}

	return nil, nil
}

func (h *handler) configToolkitContainerd(policy *nvidiav1.ClusterPolicy) (*nvidiav1.ClusterPolicy, error) {
	// check if toolkit containerd already exists by annotation
	var provider string
	var ok bool
	if provider, ok = policy.Annotations[constant.AnnotationClusterPolicyProviderKey]; ok {
		return policy, nil
	}

	detectPro, err := detector.DetectProvider(h.ctx, h.k8sClient)
	if err != nil {
		return policy, err
	}

	if detectPro != provider {
		envs := []nvidiav1.EnvVar{
			{
				Name:  "CONTAINERD_CONFIG",
				Value: getContainerdConfig(detectPro),
			},
			{
				Name:  "CONTAINERD_SOCKET",
				Value: getContainerdSocket(detectPro),
			},
			{
				Name:  "CONTAINERD_RUNTIME_CLASS",
				Value: "nvidia",
			},
			{
				Name:  "CONTAINERD_SET_AS_DEFAULT",
				Value: "true",
			},
		}
		policyObj := policy.DeepCopy()
		policyObj.Spec.Toolkit.Env = append(policy.Spec.Toolkit.Env, envs...)
		if policyObj.Annotations == nil {
			policyObj.Annotations = map[string]string{}
		}
		policyObj.Annotations[constant.AnnotationClusterPolicyProviderKey] = detectPro
		return h.policies.Update(policyObj)
	}

	return policy, nil
}

func getContainerdConfig(provider string) string {
	if strings.Contains(provider, "k3s") {
		provider = string(RuntimeK3S)
		return fmt.Sprintf("/var/lib/rancher/%s/agent/etc/containerd/config.toml.tmpl", provider)
	} else if strings.Contains(provider, "rke2") {
		provider = string(RuntimeRKE2)
		return fmt.Sprintf("/var/lib/rancher/%s/agent/etc/containerd/config.toml.tmpl", provider)
	}
	return "/etc/containerd/config.toml"
}

func getContainerdSocket(provider string) string {
	if strings.Contains(provider, "k3s") || strings.Contains(provider, "rke2") {
		return "/run/k3s/containerd/containerd.sock"
	}
	return "/run/containerd/containerd.sock"
}
