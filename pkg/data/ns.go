package data

import (
	"github.com/rancher/wrangler/v2/pkg/apply"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/llmos-ai/llmos-controller/pkg/constant"
)

func addPublicNamespace(apply apply.Apply) error {
	// add public namespace for all authenticated users
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: constant.LLMOSPublicNamespace},
	}

	return apply.
		WithDynamicLookup().
		WithSetID("add-public-ns").
		ApplyObjects(namespace)
}
