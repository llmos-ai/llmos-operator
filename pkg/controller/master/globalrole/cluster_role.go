package globalrole

import (
	"fmt"

	wrangler "github.com/rancher/wrangler/v3/pkg/name"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
)

func constructClusterRole(role *v1.GlobalRole) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: GenerateCRName(role.Name),
			Labels: map[string]string{
				defaultGlobalRoleLabelKey: "true",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(role, role.GroupVersionKind()),
			},
		},
		Rules: role.Rules,
	}
}

func constructGlobalNsRoles(role *v1.GlobalRole) []*rbacv1.Role {
	if len(role.NamespacedRules) == 0 {
		return nil
	}

	roles := make([]*rbacv1.Role, 0, len(role.NamespacedRules))
	ownerRefs := []metav1.OwnerReference{
		*metav1.NewControllerRef(role, role.GroupVersionKind()),
	}
	for ns, rules := range role.NamespacedRules {
		role := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GenerateRoleName(role.Name, ns),
				Namespace: ns,
				Labels: map[string]string{
					defaultGlobalRoleLabelKey: "true",
				},
				OwnerReferences: ownerRefs,
			},
			Rules: rules,
		}
		roles = append(roles, role)
	}
	return roles
}

func GenerateCRName(name string) string {
	return fmt.Sprintf("llmos-globalrole-%s", name)
}
func GenerateRoleName(name, ns string) string {
	return wrangler.SafeConcatName(name, ns)
}
