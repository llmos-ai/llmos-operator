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
)

type roleHandler struct {
	roleClient ctlmgmtv1.GlobalRoleClient
	roleCache  ctlmgmtv1.GlobalRoleCache
}

func BootstrapGlobalRoles(mgmt *config.Management) error {
	globalRole := mgmt.MgmtFactory.Management().V1().GlobalRole()
	h := &roleHandler{
		roleClient: globalRole,
		roleCache:  globalRole.Cache(),
	}

	roles := initDefaultRoles()
	for _, role := range roles {
		_, err := h.roleClient.Create(role)
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func initDefaultRoles() []*mgmtv1.GlobalRole {
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
							"*",
						},
						Resources: []string{
							"persistentvolumes",
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
							"ml.llmos.ai",
							"management.llmos.ai",
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
