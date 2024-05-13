package config

import (
	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	"github.com/rancher/wrangler/v2/pkg/schemes"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	mgmtv1 "github.com/llmos-ai/llmos-controller/pkg/apis/management.llmos.ai/v1"
)

var (
	localSchemeBuilder = runtime.SchemeBuilder{
		mgmtv1.AddToScheme,
		upgradev1.AddToScheme,
	}
	AddToScheme = localSchemeBuilder.AddToScheme
	Scheme      = runtime.NewScheme()
)

func init() {
	utilruntime.Must(AddToScheme(Scheme))
	utilruntime.Must(schemes.AddToScheme(Scheme))
}
