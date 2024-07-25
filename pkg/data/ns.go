package data

import (
	"github.com/rancher/wrangler/v3/pkg/apply"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
)

var defaultNSs = []string{
	constant.PublicNamespaceName,
	constant.CephSystemNamespaceName,
}

func addDefaultNamespaces(apply apply.Apply) error {
	// add default system & public namespaces
	var nss = make([]runtime.Object, 0)
	for _, ns := range defaultNSs {
		nss = append(nss, &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		})
	}

	return apply.
		WithDynamicLookup().
		WithSetID("add-default-nss").
		ApplyObjects(nss...)
}
