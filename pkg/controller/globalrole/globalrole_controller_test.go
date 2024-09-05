package globalrole

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/fake"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/utils/fakeclients"
)

var (
	adminRole = &mgmtv1.GlobalRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "admin",
		},
		Spec: mgmtv1.GlobalRoleSpec{
			Builtin:     true,
			DisplayName: "Admin",
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
	}
	userRole = &mgmtv1.GlobalRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "user",
		},
		Spec: mgmtv1.GlobalRoleSpec{
			Builtin:     true,
			DisplayName: "User",
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
	}
)

type input struct {
	key        string
	GlobalRole *mgmtv1.GlobalRole
}
type output struct {
	GlobalRole  *mgmtv1.GlobalRole
	ClusterRole *rbacv1.ClusterRole
	Role        []*rbacv1.Role
	err         error
}

func Test_OnChangeRoles(t *testing.T) {
	var testCases = []struct {
		name  string
		given input
		want  output
	}{
		{
			name: "create admin global role",
			given: input{
				key:        "admin",
				GlobalRole: adminRole,
			},
			want: output{
				ClusterRole: &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: GenerateCRName(adminRole.Name),
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
				err: nil,
			},
		},
		{
			name: "create user global role",
			given: input{
				key:        "user",
				GlobalRole: userRole,
			},
			want: output{
				ClusterRole: &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: GenerateCRName(userRole.Name),
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
				},
				Role: []*rbacv1.Role{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: GenerateRoleName(userRole.Name, constant.PublicNamespaceName),
						},
						Rules: []rbacv1.PolicyRule{
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
				err: nil,
			},
		},
	}
	for _, tc := range testCases {
		fakeClient := fake.NewSimpleClientset()
		k8sClient := k8sfake.NewSimpleClientset()
		if tc.given.GlobalRole != nil {
			var err = fakeClient.Tracker().Add(tc.given.GlobalRole)
			assert.Nil(t, err, "mock resource should add successfully")
		}

		h := &handler{
			globalRoleClient: fakeclients.GlobalRole(fakeClient.ManagementV1().GlobalRoles),
			crClient:         fakeclients.ClusterRole(k8sClient.RbacV1().ClusterRoles),
			crCache:          fakeclients.ClusterRoleCache(k8sClient.RbacV1().ClusterRoles),
			roleClient:       fakeclients.RoleClient(k8sClient.RbacV1().Roles),
			roleCache:        fakeclients.RoleCache(k8sClient.RbacV1().Roles),
		}

		var actual output
		actual.GlobalRole, actual.err = h.onChangeRoles(tc.given.key, tc.given.GlobalRole)
		assert.NoError(t, actual.err, "case %q", tc.name)
		assert.NotNil(t, actual.GlobalRole)
		if tc.want.ClusterRole != nil {
			cr, err := h.reconcileClusterRole(tc.given.GlobalRole)
			assert.NoError(t, err)
			assert.NotNil(t, cr)
			assert.Equal(t, tc.want.ClusterRole.Rules, cr.Rules)
			assert.Equal(t, tc.want.ClusterRole.Name, cr.Name)
		}

		if tc.want.Role != nil {
			roles, err := h.reconcileNamespacedRoles(tc.given.GlobalRole)
			assert.NoError(t, err)
			assert.Equal(t, len(roles), len(tc.want.Role))
			assert.Equal(t, tc.want.Role[0].Rules, roles[0].Rules)
			assert.Equal(t, tc.want.Role[0].Name, roles[0].Name)
		}
	}
}
