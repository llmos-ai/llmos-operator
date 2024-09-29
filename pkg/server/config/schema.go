package config

import (
	nvidiav1 "github.com/NVIDIA/gpu-operator/api/nvidia/v1"
	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	"github.com/rancher/wrangler/v3/pkg/schemes"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	rookv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
)

var (
	localSchemeBuilder = runtime.SchemeBuilder{
		mgmtv1.AddToScheme,
		mlv1.AddToScheme,
		upgradev1.AddToScheme,
		rayv1.AddToScheme,
		nvidiav1.AddToScheme,
		rookv1.AddToScheme,
	}
	AddToScheme = localSchemeBuilder.AddToScheme
	Scheme      = runtime.NewScheme()
)

func init() {
	utilruntime.Must(AddToScheme(Scheme))
	utilruntime.Must(schemes.AddToScheme(Scheme))
}
