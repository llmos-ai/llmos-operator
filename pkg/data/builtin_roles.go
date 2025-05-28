package data

import (
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	DefaultAdminRoleName = "admin"
	DefaultUserRoleName  = "user"
	DefaultNsOwner       = "namespace-owner"
	DefaultNsReadOnly    = "namespace-readonly"
)

func BootstrapGlobalRoles(mgmt *config.Management) error {
	globalRoles := constructDefaultGlobalRole()
	err := mgmt.Apply.WithDynamicLookup().WithSetID("apply-default-global-role-templates").ApplyObjects(globalRoles...)
	if err != nil {
		return fmt.Errorf("failed to apply built-in GlobalRoles: %v", err)
	}

	roleTemplates := constructDefaultNsRoleTemplates()
	err = mgmt.Apply.WithDynamicLookup().WithSetID("apply-default-ns-role-templates").ApplyObjects(roleTemplates...)
	if err != nil {
		return fmt.Errorf("failed to apply built-in RoleTempaltes: %v", err)
	}

	return nil
}

func constructDefaultGlobalRole() []runtime.Object {
	return []runtime.Object{
		&mgmtv1.GlobalRole{
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
		&mgmtv1.GlobalRole{
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
						"management.llmos.ai",
					},
					Resources: []string{
						"users",
						"settings",
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
					},
					Verbs: []string{
						"create",
						"get",
						"list", // list all is filtered by admin role
						"watch",
						"delete",
					},
				},
			},
			NamespacedRules: map[string][]rbacv1.PolicyRule{
				constant.LLMOSAgentsNamespaceName: {
					{
						APIGroups: []string{
							"",
						},
						Resources: []string{
							"persistentvolumeclaims",
							"configmaps",
							"services",
							"pods",
							"events",
						},
						Verbs: []string{
							"get",
							"list",
							"watch",
						},
					},
				},
				constant.PublicNamespaceName: {
					{
						APIGroups: []string{
							"",
						},
						Resources: []string{
							"persistentvolumeclaims",
							"configmaps",
							"services",
							"pods",
							"events",
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

func constructDefaultNsRoleTemplates() []runtime.Object {
	return []runtime.Object{
		&mgmtv1.RoleTemplate{
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
					// for now, we allow all actions on all resources within the namespace
					APIGroups: []string{
						"",
					},
					Resources: []string{
						"*",
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
		&mgmtv1.RoleTemplate{
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
						"namespaces",
					},
					Verbs: []string{
						"get",
					},
				},
				{
					APIGroups: []string{
						"",
					},
					Resources: []string{
						"persistentvolumeclaims",
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
