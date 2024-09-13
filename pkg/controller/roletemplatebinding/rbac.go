package roletemplatebinding

import (
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/controller/globalrole"
)

func (h *handler) getClusterRole(rtb *mgmtv1.RoleTemplateBinding) (*rbacv1.ClusterRole, error) {
	crName := globalrole.GenerateCRName(rtb.RoleTemplateRef.Name)
	return h.crCache.Get(crName)
}

func constructClusterRoleBinding(rtb *mgmtv1.RoleTemplateBinding, cr *rbacv1.ClusterRole) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: GenerateCRBName(rtb),
			Labels: map[string]string{
				roleTemplateRefNameLabelKey: rtb.RoleTemplateRef.Name,
				roleTemplateRefKindLabelKey: rtb.RoleTemplateRef.Kind,
				rtbNameLabelKey:             rtb.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(rtb, rtb.GroupVersionKind()),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     cr.Name,
		},
		Subjects: rtb.Subjects,
	}
}

func constructRole(rtb *mgmtv1.RoleTemplateBinding, rules []rbacv1.PolicyRule, ns string) *rbacv1.Role {
	roleRules := append(rules, rbacv1.PolicyRule{
		APIGroups: []string{""},
		Resources: []string{"namespaces"},
		Verbs:     []string{"get", "list", "watch"},
	})
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GenerateRoleBindingName(rtb),
			Namespace: ns,
			Labels: map[string]string{
				roleTemplateRefNameLabelKey: rtb.RoleTemplateRef.Name,
				roleTemplateRefKindLabelKey: rtb.RoleTemplateRef.Kind,
				rtbNameLabelKey:             rtb.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(rtb, rtb.GroupVersionKind()),
			},
		},
		Rules: roleRules,
	}
}

func constructRoleBinding(rtb *mgmtv1.RoleTemplateBinding, role *rbacv1.Role, ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GenerateRoleBindingName(rtb),
			Namespace: ns,
			Labels: map[string]string{
				roleTemplateRefNameLabelKey: rtb.RoleTemplateRef.Name,
				roleTemplateRefKindLabelKey: rtb.RoleTemplateRef.Kind,
				rtbNameLabelKey:             rtb.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(rtb, rtb.GroupVersionKind()),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     role.Name,
		},
		Subjects: rtb.Subjects,
	}
}

func GenerateCRBName(rtb *mgmtv1.RoleTemplateBinding) string {
	return fmt.Sprintf("llmos-globalrole-%s", rtb.Name)
}

func GenerateRoleBindingName(rtb *mgmtv1.RoleTemplateBinding) string {
	return fmt.Sprintf("llmos-namespacedrole-%s", rtb.Name)
}
