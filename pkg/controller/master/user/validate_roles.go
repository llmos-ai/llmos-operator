package user

import (
	"reflect"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/roletemplatebinding"
	"github.com/llmos-ai/llmos-operator/pkg/data"
)

func (h *handler) OnRoleTemplateBindingChanged(_ string, rtb *mgmtv1.RoleTemplateBinding) (
	*mgmtv1.RoleTemplateBinding, error) {
	if rtb == nil || rtb.DeletionTimestamp != nil {
		return nil, nil
	}

	user, err := h.findUserByRtb(rtb)
	if err != nil {
		return rtb, err
	}
	if user != nil {
		toUpdate := user.DeepCopy()
		if rtb.RoleTemplateRef.Name == data.DefaultAdminRoleName &&
			rtb.RoleTemplateRef.Kind == roletemplatebinding.GlobalRoleKindName {
			toUpdate.Status.IsAdmin = true
		}

		if !reflect.DeepEqual(user.Status, toUpdate.Status) {
			_, err = h.users.UpdateStatus(toUpdate)
			if err != nil {
				return rtb, err
			}
		}
		return rtb, nil
	}

	return rtb, nil
}

func (h *handler) OnRoleTemplateBindingDeleted(_ string, rtb *mgmtv1.RoleTemplateBinding) (
	*mgmtv1.RoleTemplateBinding, error) {
	if rtb == nil || rtb.DeletionTimestamp == nil {
		return nil, nil
	}

	user, err := h.findUserByRtb(rtb)
	if err != nil {
		return rtb, err
	}

	if user != nil {
		toUpdate := user.DeepCopy()
		if rtb.RoleTemplateRef.Name == data.DefaultAdminRoleName &&
			rtb.RoleTemplateRef.Kind == roletemplatebinding.GlobalRoleKindName {
			toUpdate.Status.IsAdmin = false
		}

		if !reflect.DeepEqual(user.Status, toUpdate.Status) {
			_, err = h.users.UpdateStatus(toUpdate)
			if err != nil {
				return rtb, err
			}
		}
		return rtb, nil
	}

	return rtb, nil
}

func (h *handler) findUserByRtb(rtb *mgmtv1.RoleTemplateBinding) (*mgmtv1.User, error) {
	userName := ""
	for _, subject := range rtb.Subjects {
		if subject.Kind == "User" {
			userName = subject.Name
			break
		}
	}

	if userName == "" {
		logrus.Warnf("no user found in roleTemplateBinding %s", rtb.Name)
		return nil, nil
	}

	// find user by name and update the admin spec
	user, err := h.userCache.Get(userName)
	if err != nil && errors.IsNotFound(err) {
		logrus.Warnf("user %s not found, but it is defined in the roleTempalteBinding %s", userName, rtb.Name)
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return user, nil
}
