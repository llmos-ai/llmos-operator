package indexeres

import (
	"context"

	steve "github.com/rancher/steve/pkg/server"
	rbacv1 "k8s.io/api/rbac/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	sconfig "github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	UserNameIndex               = "management.llmos.ai/user-username-index"
	TokenNameIndex              = "management.llmos.ai/token-name-index"
	ClusterRoleBindingNameIndex = "management.llmos.ai/crb-by-role-and-subject-index"
)

func Register(ctx context.Context, _ *steve.Controllers, _ sconfig.Options) error {
	scaled := sconfig.ScaledWithContext(ctx)
	mgmt := scaled.Management
	crbInformer := mgmt.RbacFactory.Rbac().V1().ClusterRoleBinding().Cache()
	userInformer := mgmt.MgmtFactory.Management().V1().User().Cache()
	tokenInformer := mgmt.MgmtFactory.Management().V1().Token().Cache()

	crbInformer.AddIndexer(ClusterRoleBindingNameIndex, rbByRoleAndSubject)
	userInformer.AddIndexer(UserNameIndex, indexUserByUsername)
	tokenInformer.AddIndexer(TokenNameIndex, tokenKeyIndexer)
	return nil
}

func indexUserByUsername(obj *mgmtv1.User) ([]string, error) {
	return []string{obj.Spec.Username}, nil
}

func tokenKeyIndexer(token *mgmtv1.Token) ([]string, error) {
	return []string{token.Name}, nil
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
