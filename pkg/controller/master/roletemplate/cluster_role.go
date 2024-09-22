package roletemplate

import (
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
)

func constructClusterRole(rt *mgmtv1.RoleTemplate) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: GenerateRoleTemplateName(rt.Name),
			Labels: map[string]string{
				LabelAuthRoleTemplateNameKey: rt.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(rt, rt.GroupVersionKind()),
			},
		},
		Rules: rt.Rules,
	}
}

func GenerateRoleTemplateName(name string) string {
	return fmt.Sprintf("llmos-roletemplate-%s", name)
}
