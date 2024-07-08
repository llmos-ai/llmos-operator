package data

import (
	"fmt"

	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	mgmtv1 "github.com/llmos-ai/llmos-controller/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-controller/pkg/auth"
	"github.com/llmos-ai/llmos-controller/pkg/server/config"
)

const (
	defaultAdminLabelKey   = "management.llmos.ai/default-admin"
	defaultAdminLabelValue = "true"
	defaultAdminPassword   = "password"
)

var defaultAdminLabel = map[string]string{
	defaultAdminLabelKey: defaultAdminLabelValue,
}

func BootstrapDefaultAdmin(mgmt *config.Management) error {
	set := labels.Set(defaultAdminLabel)
	admins, err := mgmt.MgmtFactory.Management().V1().User().List(metav1.ListOptions{LabelSelector: set.String()})
	if err != nil {
		return err
	}

	if len(admins.Items) > 0 {
		logrus.Info("Default admin already exist, skip creating")
		return nil
	}

	// admin user not exist, attempt to create the default admin user
	hash, err := auth.HashPassword(defaultAdminPassword)
	if err != nil {
		return err
	}

	user, err := mgmt.MgmtFactory.Management().V1().User().Create(&mgmtv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "user-",
			Labels:       defaultAdminLabel,
		},
		Spec: mgmtv1.UserSpec{
			DisplayName: "Default Admin",
			Username:    "admin",
			Password:    hash,
			IsAdmin:     true,
			IsActive:    true,
		},
	})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	_, err = mgmt.RbacFactory.Rbac().V1().ClusterRoleBinding().Create(
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "default-admin-",
				Labels: map[string]string{
					defaultAdminLabelKey: defaultAdminLabelValue,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: mgmtv1.SchemeGroupVersion.String(),
						Kind:       "User",
						Name:       user.Name,
						UID:        user.UID,
					},
				},
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:     "User",
					APIGroup: rbacv1.GroupName,
					Name:     user.Name,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			},
		})
	if err != nil {
		return fmt.Errorf("failed to create default admin cluster role binding: %v", err)
	}
	logrus.Info("successfully created default admin user and cluster role binding")

	return nil
}
