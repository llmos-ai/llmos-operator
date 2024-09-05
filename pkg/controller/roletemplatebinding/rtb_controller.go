package roletemplatebinding

import (
	"context"
	"fmt"

	ctlrbacv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/rbac/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	rtbOnChangeName        = "roleTemplateBinding.onChange"
	globalRoleNameLabelKey = "auth.management.llmos.ai/global-role-name"
	rtbNameLabelKey        = "auth.management.llmos.ai/role-template-binding-name"

	GlobalRoleKindName = "GlobalRole"
)

type handler struct {
	crClient          ctlrbacv1.ClusterRoleClient
	crCache           ctlrbacv1.ClusterRoleCache
	crbClient         ctlrbacv1.ClusterRoleBindingClient
	crbCache          ctlrbacv1.ClusterRoleBindingCache
	roleClient        ctlrbacv1.RoleClient
	roleCache         ctlrbacv1.RoleCache
	roleBindingClient ctlrbacv1.RoleBindingClient
	roleBindingCache  ctlrbacv1.RoleBindingCache
	grClient          ctlmgmtv1.GlobalRoleClient
	grCache           ctlmgmtv1.GlobalRoleCache
}

func Register(ctx context.Context, mgmt *config.Management) error {
	rtb := mgmt.MgmtFactory.Management().V1().RoleTemplateBinding()
	crs := mgmt.RbacFactory.Rbac().V1().ClusterRole()
	crb := mgmt.RbacFactory.Rbac().V1().ClusterRoleBinding()
	roles := mgmt.RbacFactory.Rbac().V1().Role()
	gr := mgmt.MgmtFactory.Management().V1().GlobalRole()
	rb := mgmt.RbacFactory.Rbac().V1().RoleBinding()

	h := &handler{
		crClient:          crs,
		crCache:           crs.Cache(),
		crbClient:         crb,
		crbCache:          crb.Cache(),
		roleClient:        roles,
		roleCache:         roles.Cache(),
		roleBindingClient: rb,
		roleBindingCache:  rb.Cache(),
		grClient:          gr,
		grCache:           gr.Cache(),
	}
	rtb.OnChange(ctx, rtbOnChangeName, h.onChange)
	return nil
}

// onChange watches RoleTemplateBinding changes and creates/updates the corresponding
// ClusterRole/Role and ClusterRoleBinding/RoleBinding
func (h *handler) onChange(_ string, rtb *mgmtv1.RoleTemplateBinding) (*mgmtv1.RoleTemplateBinding, error) {
	if rtb == nil || rtb.DeletionTimestamp != nil {
		return rtb, nil
	}

	gr, err := h.grCache.Get(rtb.RoleTemplateRef.Name)
	if err != nil {
		return rtb, fmt.Errorf("failed to get global role %s: %v", rtb.RoleTemplateRef.Name, err)
	}

	refKind := rtb.RoleTemplateRef.Kind
	switch refKind {
	case GlobalRoleKindName:
		// create cluster roleBinding
		if err := h.reconcileClusterRoleBinding(rtb, gr); err != nil {
			return rtb, err
		}

		// create namespaced role and roleBinding
		if err := h.reconcileNamespacedRoles(rtb, gr); err != nil {
			return rtb, err
		}
		return rtb, nil
	default:
		logrus.Errorf("unsupported roleTemplateRef kind %s", refKind)
		return rtb, nil
	}
}

func (h *handler) reconcileClusterRoleBinding(rtb *mgmtv1.RoleTemplateBinding, gr *mgmtv1.GlobalRole) error {
	cr, err := h.getClusterRole(rtb)
	if err != nil {
		return err
	}

	crb := constructClusterRoleBinding(rtb, gr, cr)
	foundCrb, err := h.crbCache.Get(crb.Name)
	if err != nil && errors.IsNotFound(err) {
		logrus.Debugf("creating cluster role binding %+v", crb)
		if _, err = h.crbClient.Create(crb); err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	if foundCrb != nil {
		logrus.Debugf("cluster rolbe binding %s already exist, nothing to change", foundCrb.Name)
	}
	return nil
}

func (h *handler) reconcileNamespacedRoles(rtb *mgmtv1.RoleTemplateBinding, gr *mgmtv1.GlobalRole) error {
	for ns, rules := range gr.NamespacedRules {
		role := constructRole(rtb, gr, rules, ns)
		_, err := h.roleCache.Get(ns, role.Name)
		if err != nil && errors.IsNotFound(err) {
			logrus.Debugf("creating role %s:%s of roleTemplateBinding %s", role.Name, role.Namespace, rtb.Name)
			role, err = h.roleClient.Create(role)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		rb := constructRoleBinding(rtb, gr, role, ns)
		logrus.Debugf("cluster role binding %s", rb.Name)
		_, err = h.roleBindingCache.Get(ns, rb.Name)
		if err != nil && errors.IsNotFound(err) {
			logrus.Debugf("creating roleBinding %s:%s of roleTemplateBinding %s", rb.Name, rb.Namespace, rtb.Name)
			if _, err = h.roleBindingClient.Create(rb); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}
