package indexeres

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	UserNameIndex               = "management.llmos.ai/user-username-index"
	UserUIDIndex                = "management.llmos.ai/user-uid-index"
	ClusterRoleBindingNameIndex = "management.llmos.ai/crb-by-role-and-subject-index"
)

func Register(_ context.Context, mgmt *config.Management) error {
	userInformer := mgmt.MgmtFactory.Management().V1().User().Cache()
	userInformer.AddIndexer(UserNameIndex, indexUserByUsername)
	userInformer.AddIndexer(UserUIDIndex, indexUserByUID)
	crbInformer := mgmt.RbacFactory.Rbac().V1().ClusterRoleBinding().Cache()
	crbInformer.AddIndexer(ClusterRoleBindingNameIndex, rbByRoleAndSubject)
	return nil
}

func indexUserByUsername(obj *mgmtv1.User) ([]string, error) {
	return []string{obj.Spec.Username}, nil
}

func indexUserByUID(obj *mgmtv1.User) ([]string, error) {
	return []string{string(obj.UID)}, nil
}

func rbByRoleAndSubject(obj *rbacv1.ClusterRoleBinding) ([]string, error) {
	keys := make([]string, len(obj.Subjects))
	for _, s := range obj.Subjects {
		keys = append(keys, GetCrbKey(obj.RoleRef.Name, s))
	}
	return keys, nil
}

func GetCrbKey(roleName string, subject rbacv1.Subject) string {
	return roleName + "." + subject.Kind + "." + subject.Name
}
