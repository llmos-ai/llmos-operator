package user

import (
	"fmt"

	"github.com/oneblock-ai/webhook/pkg/server/admission"
	"github.com/sirupsen/logrus"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/auth/tokens"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
)

type mutator struct {
	admission.DefaultMutator
}

var _ admission.Mutator = &mutator{}

func NewMutator() admission.Mutator {
	return &mutator{}
}

func (m *mutator) Create(_ *admission.Request, newObj runtime.Object) (admission.Patch, error) {
	user := newObj.(*mgmtv1.User)
	logrus.Infof("[webhook mutating]user %s is created", user.Name)

	if user.Spec.Password == "" {
		return nil, fmt.Errorf("password can't be empty")
	}

	patchOps := make([]admission.PatchOp, 0)

	patchOps = append(patchOps, patchLabels(user.Labels))

	// skip default admin password hash
	if user.Labels != nil && user.Labels[constant.DefaultAdminLabelKey] == "true" {
		logrus.Infof("skip default admin password hash")
	} else {
		// hash password
		passPatch, err := patchPassword(user.Spec.Password)
		if err != nil {
			return nil, err
		}
		patchOps = append(patchOps, passPatch)
	}

	return patchOps, nil
}

func patchLabels(labels map[string]string) admission.PatchOp {
	if labels == nil {
		labels = map[string]string{}
	}
	labels["llmos.ai/creator"] = "llmos-operator"
	return admission.PatchOp{
		Op:    admission.PatchOpReplace,
		Path:  "/metadata/labels",
		Value: labels,
	}
}

func patchPassword(password string) (admission.PatchOp, error) {
	hash, err := tokens.HashPassword(password)
	if err != nil {
		return admission.PatchOp{}, err
	}
	return admission.PatchOp{
		Op:    admission.PatchOpReplace,
		Path:  "/spec/password",
		Value: hash,
	}, nil
}

func (m *mutator) Update(_ *admission.Request, oldObj, newObj runtime.Object) (admission.Patch, error) {
	oldUSer := oldObj.(*mgmtv1.User)
	newUser := newObj.(*mgmtv1.User)
	logrus.Debugf("newUser %s is updated", newUser.Name)

	patchOps := make([]admission.PatchOp, 0)

	if (oldUSer.Spec.Password != newUser.Spec.Password) && newUser.Spec.Password != "" {
		logrus.Debugf("updating new password")
		passPatch, err := patchPassword(newUser.Spec.Password)
		if err != nil {
			return nil, err
		}

		patchOps = append(patchOps, passPatch)
	}

	return patchOps, nil
}

func (m *mutator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"users"},
		Scope:      admissionregv1.ClusterScope,
		APIGroup:   mgmtv1.SchemeGroupVersion.Group,
		APIVersion: mgmtv1.SchemeGroupVersion.Version,
		ObjectType: &mgmtv1.User{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}
