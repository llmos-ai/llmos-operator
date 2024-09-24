package data

import (
	"fmt"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/auth/tokens"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

const (
	defaultAdminLabelValue = "true"
)

var defaultAdminLabel = map[string]string{
	constant.DefaultAdminLabelKey: defaultAdminLabelValue,
}

type handler struct {
	userClient ctlmgmtv1.UserClient
	rtbClient  ctlmgmtv1.RoleTemplateBindingClient
}

func BootstrapDefaultAdmin(mgmt *config.Management) error {
	h := &handler{
		userClient: mgmt.MgmtFactory.Management().V1().User(),
		rtbClient:  mgmt.MgmtFactory.Management().V1().RoleTemplateBinding(),
	}

	users, err := h.userClient.List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, user := range users.Items {
		if hasAdminUser(user) {
			logrus.Debugf("Default admin %s already exist, skip creating", user.Spec.Username)
			return nil
		}
	}

	initPassword, err := utils.GenerateToken()
	if err != nil {
		return fmt.Errorf("failed to generate init password: %v", err)
	}

	logrus.Infof("Bootstrap password: %s", initPassword)
	initPassSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "llmos-bootstrap-passwd",
			Namespace: constant.SystemNamespaceName,
		},
		Data: map[string][]byte{
			"password": []byte(initPassword),
		},
	}

	err = mgmt.Apply.WithDynamicLookup().WithSetID("apply-admin-init-password").ApplyObjects(initPassSecret)
	if err != nil {
		return fmt.Errorf("failed to create init password secret: %v", err)
	}

	hash, err := tokens.HashPassword(initPassword)
	if err != nil {
		return fmt.Errorf("failed to hash init password: %v", err)
	}

	user := &mgmtv1.User{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "user-",
			Labels:       defaultAdminLabel,
		},
		Spec: mgmtv1.UserSpec{
			DisplayName: "Default Admin",
			Username:    "admin",
			Password:    hash,
			Active:      true,
		},
	}

	user, err = h.userClient.Create(user)
	if err != nil {
		return err
	}

	rtb := constructRoleTemplateBinding(user)
	if _, err := h.rtbClient.Create(rtb); err != nil {
		return fmt.Errorf("failed to create default admin role template binding: %v", err)
	}

	logrus.Info("bootstrap default admin successfully")
	return nil
}

func constructRoleTemplateBinding(user *mgmtv1.User) *mgmtv1.RoleTemplateBinding {
	return &mgmtv1.RoleTemplateBinding{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "rtb-",
			Labels: map[string]string{
				constant.DefaultAdminLabelKey: defaultAdminLabelValue,
				tokens.LabelAuthUserId:        user.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(user, user.GroupVersionKind()),
			},
		},
		RoleTemplateRef: mgmtv1.RoleTemplateRef{
			APIGroup: mgmtv1.SchemeGroupVersion.Group,
			Kind:     "GlobalRole",
			Name:     DefaultAdminRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     "User",
				Name:     user.Name,
			},
		},
	}
}

func hasAdminUser(user mgmtv1.User) bool {
	if user.Status.IsAdmin {
		return true
	}

	if user.Labels != nil && user.Labels[constant.DefaultAdminLabelKey] == defaultAdminLabelValue {
		return true
	}

	return false
}
