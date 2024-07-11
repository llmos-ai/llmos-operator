package user

import (
	"fmt"

	"github.com/oneblock-ai/webhook/pkg/server/admission"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	managementv1 "github.com/llmos-ai/llmos-controller/pkg/apis/management.llmos.ai/v1"
	ctlmanagementv1 "github.com/llmos-ai/llmos-controller/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-controller/pkg/webhook/config"
)

type validator struct {
	admission.DefaultValidator
	userCache ctlmanagementv1.UserCache
}

var _ admission.Validator = &validator{}

func NewValidator(mgmt *config.Management) admission.Validator {
	return &validator{
		userCache: mgmt.MgmtFactory.Management().V1().User().Cache(),
	}
}

func (v *validator) Create(_ *admission.Request, newObj runtime.Object) error {
	user := newObj.(*managementv1.User)

	users, err := v.userCache.List(labels.Everything())
	if err != nil {
		return err
	}

	for _, u := range users {
		if u.Spec.DisplayName == user.Spec.DisplayName {
			return fmt.Errorf("the display name %s is already taken", user.Spec.DisplayName)
		}
	}

	return nil
}

func (v *validator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"users"},
		Scope:      admissionregv1.ClusterScope,
		APIGroup:   managementv1.SchemeGroupVersion.Group,
		APIVersion: managementv1.SchemeGroupVersion.Version,
		ObjectType: &managementv1.User{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
		},
	}
}
