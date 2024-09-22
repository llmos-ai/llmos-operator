package roletemplate

import (
	"context"
	"fmt"
	"reflect"
	"time"

	ctlrbacv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/rbac/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	cond "github.com/llmos-ai/llmos-operator/pkg/utils/condition"
)

const (
	roleTemplateOnChange         = "roleTemplate.onChangeRoles"
	LabelAuthRoleTemplateNameKey = "auth.management.llmos.ai/role-template-name"
)

type handler struct {
	roleTemplateClient ctlmgmtv1.RoleTemplateClient
	roleTemplateCache  ctlmgmtv1.RoleTemplateCache
	crClient           ctlrbacv1.ClusterRoleClient
	crCache            ctlrbacv1.ClusterRoleCache
}

func Register(ctx context.Context, mgmt *config.Management, _ config.Options) error {
	roleTemplates := mgmt.MgmtFactory.Management().V1().RoleTemplate()
	crs := mgmt.RbacFactory.Rbac().V1().ClusterRole()

	h := &handler{
		roleTemplateClient: roleTemplates,
		roleTemplateCache:  roleTemplates.Cache(),
		crClient:           crs,
		crCache:            crs.Cache(),
	}
	roleTemplates.OnChange(ctx, roleTemplateOnChange, h.onChangeRoles)
	return nil
}

func (h *handler) onChangeRoles(_ string, rt *mgmtv1.RoleTemplate) (*mgmtv1.RoleTemplate, error) {
	if rt == nil || rt.DeletionTimestamp != nil {
		return nil, nil
	}

	// init status first
	if rt.Status.State == "" {
		toUpdate := rt.DeepCopy()
		mgmtv1.ClusterRoleExists.CreateUnknownIfNotExists(toUpdate)
		toUpdate.Status.State = "InProgress"
		return h.roleTemplateClient.UpdateStatus(toUpdate)
	}

	cr, err := h.reconcileClusterRole(rt)
	if err != nil {
		return h.updateErrorStatus(rt, mgmtv1.ClusterRoleExists, err)
	}

	if err = h.updateStatus(rt, cr); err != nil {
		return rt, err
	}

	return rt, nil
}

func (h *handler) reconcileClusterRole(rt *mgmtv1.RoleTemplate) (*rbacv1.ClusterRole, error) {
	cr := constructClusterRole(rt)
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
		toUpdate.Labels[LabelAuthRoleTemplateNameKey] = rt.Name
		return h.crClient.Update(toUpdate)
	}

	return cr, nil
}

func (h *handler) updateStatus(rt *mgmtv1.RoleTemplate, cr *rbacv1.ClusterRole) error {
	if cr == nil {
		return nil
	}

	toUpdate := rt.DeepCopy()
	mgmtv1.ClusterRoleExists.Message(toUpdate, fmt.Sprintf("%s created", cr.Name))
	mgmtv1.ClusterRoleExists.SetStatus(toUpdate, "True")
	mgmtv1.ClusterRoleExists.Reason(toUpdate, "Created")

	toUpdate.Status.LastUpdate = time.Now().Format(time.RFC3339)
	toUpdate.Status.ObservedGeneration = toUpdate.ObjectMeta.Generation
	toUpdate.Status.State = "Complete"

	if !reflect.DeepEqual(toUpdate.Status, rt.Status) {
		if _, err := h.roleTemplateClient.UpdateStatus(toUpdate); err != nil {
			return err
		}
	}

	return nil
}

func (h *handler) updateErrorStatus(rt *mgmtv1.RoleTemplate, cond cond.Cond, err error) (
	*mgmtv1.RoleTemplate, error) {
	toUpdate := rt.DeepCopy()
	cond.SetError(toUpdate, "Error", err)
	toUpdate.Status.State = "Error"
	return h.roleTemplateClient.UpdateStatus(toUpdate)
}
