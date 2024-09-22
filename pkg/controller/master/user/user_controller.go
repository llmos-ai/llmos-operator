package user

import (
	"context"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/auth/tokens"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"

	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	userOnChangeName                = "user.onChange"
	userRoleTemplateBindingOnChange = "user.roleTemplateBindingOnChange"
	userRoleTemplateBindingOnRemove = "user.roleTemplateBindingOnRemove"
	userOnRemoveName                = "user.onRemove"
)

type handler struct {
	users     ctlmgmtv1.UserClient
	userCache ctlmgmtv1.UserCache
	rtbClient ctlmgmtv1.RoleTemplateBindingClient
	rtbCache  ctlmgmtv1.RoleTemplateBindingCache
}

func Register(ctx context.Context, management *config.Management, _ config.Options) error {
	users := management.MgmtFactory.Management().V1().User()
	rtb := management.MgmtFactory.Management().V1().RoleTemplateBinding()
	h := &handler{
		users:     users,
		userCache: users.Cache(),
		rtbClient: rtb,
		rtbCache:  rtb.Cache(),
	}

	users.OnChange(ctx, userOnChangeName, h.OnChanged)
	users.OnRemove(ctx, userOnRemoveName, h.OnDelete)
	rtb.OnChange(ctx, userRoleTemplateBindingOnChange, h.OnRoleTemplateBindingChanged)
	rtb.OnRemove(ctx, userRoleTemplateBindingOnRemove, h.OnRoleTemplateBindingDeleted)
	return nil
}

// OnChanged reconcile the user status and add user clusterRole and clusterRoleBinding if needed
func (h *handler) OnChanged(_ string, user *mgmtv1.User) (*mgmtv1.User, error) {
	if user == nil || user.DeletionTimestamp != nil {
		return user, nil
	}

	toUpdate := user.DeepCopy()
	if toUpdate.Labels == nil {
		toUpdate.Labels = map[string]string{}
	}
	toUpdate.Labels[constant.LabelManagementUsernameKey] = user.Spec.Username

	if !reflect.DeepEqual(user, toUpdate) {
		_, err := h.users.Update(toUpdate)
		if err != nil {
			return user, err
		}
	}

	return h.updateStatus(user, toUpdate)
}

func (h *handler) updateStatus(user *mgmtv1.User, toUpdate *mgmtv1.User) (*mgmtv1.User, error) {
	toUpdate.Status.IsActive = toUpdate.Spec.Active
	if !reflect.DeepEqual(user.Status, toUpdate.Status) {
		toUpdate.Status.LastUpdateTime = metav1.Now().Format(constant.TimeLayout)
		return h.users.UpdateStatus(toUpdate)
	}
	return nil, nil
}

// OnDelete helps to remove the user's roleTemplateBindings which is bound to the user on init
func (h *handler) OnDelete(_ string, user *mgmtv1.User) (*mgmtv1.User, error) {
	if user == nil || user.DeletionTimestamp == nil {
		return nil, nil
	}

	selector := labels.SelectorFromSet(map[string]string{
		tokens.LabelAuthUserId: user.Name,
	})
	rtbs, err := h.rtbCache.List(selector)
	if err != nil {
		return user, err
	}

	for _, rtb := range rtbs {
		if err = h.rtbClient.Delete(rtb.Name, &metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return user, err
		}
	}

	return nil, nil
}
