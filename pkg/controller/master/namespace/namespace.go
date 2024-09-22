package namespace

import (
	"context"

	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	LabelAuthNamespaceIdKey = "auth.management.llmos.ai/namespace-id"
	nsOnRemove              = "namespace.onRemove"
)

type handler struct {
	nsClient  ctlcorev1.NamespaceClient
	rtbClient ctlmgmtv1.RoleTemplateBindingClient
	rtbCache  ctlmgmtv1.RoleTemplateBindingCache
}

func Register(ctx context.Context, mgmt *config.Management, _ config.Options) error {
	ns := mgmt.CoreFactory.Core().V1().Namespace()
	rtb := mgmt.MgmtFactory.Management().V1().RoleTemplateBinding()
	h := &handler{
		nsClient:  ns,
		rtbClient: rtb,
		rtbCache:  rtb.Cache(),
	}

	ns.OnRemove(ctx, nsOnRemove, h.OnRemove)

	return nil
}

func (h *handler) OnRemove(_ string, ns *corev1.Namespace) (*corev1.Namespace, error) {
	if ns == nil || ns.DeletionTimestamp == nil {
		return nil, nil
	}

	rtbs, err := h.rtbCache.List(labels.SelectorFromSet(map[string]string{
		LabelAuthNamespaceIdKey: ns.Name,
	}))

	if err != nil {
		return nil, err
	}

	for _, rtb := range rtbs {
		if err = h.rtbClient.Delete(rtb.Name, &metav1.DeleteOptions{}); err != nil {
			return nil, err
		}
	}

	return ns, nil
}
