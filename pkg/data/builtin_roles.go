package data

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	DefaultAdminRoleName = "admin"
	DefaultUserRoleName  = "user"
	DefaultNsOwner       = "namespace-owner"
	DefaultNsReadOnly    = "namespace-readonly"
)

type roleHandler struct {
	globalRoleClient ctlmgmtv1.GlobalRoleClient
	globalRoleCache  ctlmgmtv1.GlobalRoleCache
	rtClient         ctlmgmtv1.RoleTemplateClient
	rtCache          ctlmgmtv1.RoleTemplateCache
}

func BootstrapGlobalRoles(mgmt *config.Management) error {
	globalRole := mgmt.MgmtFactory.Management().V1().GlobalRole()
	roleTemplate := mgmt.MgmtFactory.Management().V1().RoleTemplate()
	h := &roleHandler{
		globalRoleClient: globalRole,
		globalRoleCache:  globalRole.Cache(),
		rtClient:         roleTemplate,
		rtCache:          roleTemplate.Cache(),
	}

	globalRoles := constructDefaultGlobalRole()
	for _, role := range globalRoles {
		_, err := h.globalRoleClient.Create(role)
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	roleTemplates := constructDefaultRoleTemplates()
	for _, rt := range roleTemplates {
		_, err := h.rtClient.Create(rt)
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func constructDefaultGlobalRole() []*mgmtv1.GlobalRole {
	return []*mgmtv1.GlobalRole{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: DefaultAdminRoleName,
			},
			Spec: mgmtv1.GlobalRoleSpec{
				DisplayName:    "Admin",
				Builtin:        true,
				NewUserDefault: false,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{
						"*",
					},
					Resources: []string{
						"*",
					},
					Verbs: []string{
						"*",
					},
				},
				{
					NonResourceURLs: []string{
						"*",
					},
					Verbs: []string{
						"*",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: DefaultUserRoleName,
			},
			Spec: mgmtv1.GlobalRoleSpec{
				DisplayName:    "User",
				Builtin:        true,
				NewUserDefault: true,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{
						"",
					},
					Resources: []string{
						"nodes",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{
						"management.llmos.ai",
					},
					Resources: []string{
						"tokens",
						"users",
						"settings",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
			},
			NamespacedRules: map[string][]rbacv1.PolicyRule{
				constant.PublicNamespaceName: {
					{
						APIGroups: []string{
							"",
						},
						Resources: []string{
							"persistentvolumes",
						},
						Verbs: []string{
							"get",
							"list",
							"watch",
						},
					},
					{
						APIGroups: []string{
							"",
						},
						Resources: []string{
							"persistetnvolumeclaims",
						},
						Verbs: []string{
							"*",
						},
					},
					{
						APIGroups: []string{
							"ml.llmos.ai",
							"management.llmos.ai",
							"ray.io",
						},
						Resources: []string{
							"*",
						},
						Verbs: []string{
							"get",
							"list",
							"watch",
						},
					},
				},
			},
		},
	}
}

func constructDefaultRoleTemplates() []*mgmtv1.RoleTemplate {
	return []*mgmtv1.RoleTemplate{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: DefaultNsOwner,
			},
			Spec: mgmtv1.RoleTemplateSpec{
				DisplayName:         "Namespace Owner",
				Builtin:             true,
				NewNamespaceDefault: true,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{
						"",
					},
					Resources: []string{
						"persistentvolumes",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{
						"",
					},
					Resources: []string{
						"persistetnvolumeclaims",
					},
					Verbs: []string{
						"*",
					},
				},
				{
					APIGroups: []string{
						"storage.k8s.io",
					},
					Resources: []string{
						"storageclasses",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{
						"apiregistration.k8s.io",
					},
					Resources: []string{
						"apiservices",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{
						"metrics.k8s.io",
					},
					Resources: []string{
						"pods",
					},
					Verbs: []string{
						"*",
					},
				},
				{
					APIGroups: []string{
						"ml.llmos.ai",
						"management.llmos.ai",
						"ray.io",
					},
					Resources: []string{
						"*",
					},
					Verbs: []string{
						"*",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: DefaultNsReadOnly,
			},
			Spec: mgmtv1.RoleTemplateSpec{
				DisplayName:         "Namespace Read-Only",
				Builtin:             true,
				NewNamespaceDefault: false,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{
						"",
					},
					Resources: []string{
						"persistentvolumes",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{
						"",
					},
					Resources: []string{
						"persistetnvolumeclaims",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{
						"storage.k8s.io",
					},
					Resources: []string{
						"storageclasses",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{
						"apiregistration.k8s.io",
					},
					Resources: []string{
						"apiservices",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{
						"metrics.k8s.io",
					},
					Resources: []string{
						"pods",
					},
					Verbs: []string{
						"*",
					},
				},
				{
					APIGroups: []string{
						"ml.llmos.ai",
						"management.llmos.ai",
						"ray.io",
					},
					Resources: []string{
						"*",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
					},
				},
			},
		},
	}
}
