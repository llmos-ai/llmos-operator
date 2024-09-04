package globalrole

import (
	"context"
	"fmt"
	"reflect"
	"time"

	ctlrbacv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/rbac/v1"
	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	globalRoleOnChangeName    = "globalRole.onChangeRoles"
	defaultGlobalRoleLabelKey = "auth.management.llmos.ai/default-global-role"
)

type handler struct {
	globalRoleClient ctlmgmtv1.GlobalRoleClient
	globalRoleCache  ctlmgmtv1.GlobalRoleCache
	crClient         ctlrbacv1.ClusterRoleClient
	crCache          ctlrbacv1.ClusterRoleCache
	roleClient       ctlrbacv1.RoleClient
	roleCache        ctlrbacv1.RoleCache
}

func Register(ctx context.Context, mgmt *config.Management) error {
	globalRoles := mgmt.MgmtFactory.Management().V1().GlobalRole()
	crs := mgmt.RbacFactory.Rbac().V1().ClusterRole()
	roles := mgmt.RbacFactory.Rbac().V1().Role()

	h := &handler{
		globalRoleClient: globalRoles,
		globalRoleCache:  globalRoles.Cache(),
		crClient:         crs,
		crCache:          crs.Cache(),
		roleClient:       roles,
		roleCache:        roles.Cache(),
	}
	globalRoles.OnChange(ctx, globalRoleOnChangeName, h.onChangeRoles)
	return nil
}

func (h *handler) onChangeRoles(_ string, globalRole *mgmtv1.GlobalRole) (*mgmtv1.GlobalRole, error) {
	if globalRole == nil || globalRole.DeletionTimestamp != nil {
		return nil, nil
	}

	cr, err := h.reconcileClusterRole(globalRole)
	if err != nil {
		return globalRole, err
	}

	roles, err := h.reconcileNamespacedRoles(globalRole)
	if err != nil {
		return globalRole, err
	}

	if err = h.updateStatus(globalRole, cr, roles); err != nil {
		return globalRole, err
	}

	return globalRole, nil
}

func (h *handler) reconcileClusterRole(globalRole *mgmtv1.GlobalRole) (*rbacv1.ClusterRole, error) {
	cr := constructClusterRole(globalRole)
	foundCR, err := h.crCache.Get(cr.Name)
	if err != nil && errors.IsNotFound(err) {
		return h.crClient.Create(cr)
	} else if err != nil {
		return nil, err
	}

	if !reflect.DeepEqual(foundCR.Rules, cr.Rules) {
		toUpdate := foundCR.DeepCopy()
		toUpdate.Rules = cr.Rules
		if toUpdate.Labels == nil {
			toUpdate.Labels = map[string]string{}
		}
		toUpdate.Labels[defaultGlobalRoleLabelKey] = "true"
		return h.crClient.Update(toUpdate)
	}

	return cr, nil
}

func (h *handler) reconcileNamespacedRoles(globalRole *mgmtv1.GlobalRole) ([]*rbacv1.Role, error) {
	roles := constructRoles(globalRole)
	if len(roles) == 0 {
		return nil, nil
	}
	for _, role := range roles {
		foundRole, err := h.roleCache.Get(role.Namespace, role.Name)
		if err != nil && errors.IsNotFound(err) {
			if _, err = h.roleClient.Create(role); err != nil {
				return nil, err
			}
			continue
		} else if err != nil {
			return nil, err
		}

		if !reflect.DeepEqual(foundRole.Rules, role.Rules) {
			toUpdate := foundRole.DeepCopy()
			toUpdate.Rules = role.Rules
			if toUpdate.Labels == nil {
				toUpdate.Labels = map[string]string{}
			}
			toUpdate.Labels[defaultGlobalRoleLabelKey] = "true"
			if _, err = h.roleClient.Update(toUpdate); err != nil {
				return nil, err
			}
		}
	}
	return roles, nil
}

func (h *handler) updateStatus(globalRole *mgmtv1.GlobalRole, cr *rbacv1.ClusterRole, roles []*rbacv1.Role) error {
	if cr == nil && (len(roles) == 0) {
		return nil
	}

	toUpdate := globalRole.DeepCopy()
	roleNames := ""
	for _, role := range roles {
		roleNames += fmt.Sprintf("%s ", role.Name)
	}

	logrus.Debugf("cr name %s", cr.Name)
	mgmtv1.ClusterRoleExists.Message(toUpdate, fmt.Sprintf("%s created", cr.Name))
	mgmtv1.ClusterRoleExists.SetStatus(toUpdate, "True")

	if len(roles) > 0 {
		mgmtv1.NamespacedRoleExists.Message(toUpdate, fmt.Sprintf("%s created", roleNames))
		mgmtv1.NamespacedRoleExists.SetStatus(toUpdate, "True")
	}
	toUpdate.Status.LastUpdate = time.Now().Format(time.RFC3339)
	toUpdate.Status.ObservedGeneration = toUpdate.ObjectMeta.Generation
	toUpdate.Status.State = "Complete"

	if !reflect.DeepEqual(toUpdate.Status, globalRole.Status) {
		if _, err := h.globalRoleClient.UpdateStatus(toUpdate); err != nil {
			return err
		}
	}

	return nil
}
