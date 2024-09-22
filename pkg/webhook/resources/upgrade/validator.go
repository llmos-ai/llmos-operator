package upgrade

import (
	"fmt"
	"strings"

	"github.com/oneblock-ai/webhook/pkg/server/admission"
	ctlappsv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/apps/v1"
	ctlcorev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlhelmv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/helm.cattle.io/v1"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/utils/condition"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/config"
	werror "github.com/llmos-ai/llmos-operator/pkg/webhook/error"
)

type validator struct {
	admission.DefaultValidator

	releaseName     string
	upgradeCache    ctlmgmtv1.UpgradeCache
	versionCache    ctlmgmtv1.VersionCache
	addonCache      ctlmgmtv1.ManagedAddonCache
	settingCache    ctlmgmtv1.SettingCache
	nodeCache       ctlcorev1.NodeCache
	deploymentCache ctlappsv1.DeploymentCache
	helmChartCache  ctlhelmv1.HelmChartCache
}

var _ admission.Validator = &validator{}

func NewValidator(mgmt *config.Management) admission.Validator {
	return &validator{
		releaseName:     mgmt.ReleaseName,
		upgradeCache:    mgmt.MgmtFactory.Management().V1().Upgrade().Cache(),
		versionCache:    mgmt.MgmtFactory.Management().V1().Version().Cache(),
		addonCache:      mgmt.MgmtFactory.Management().V1().ManagedAddon().Cache(),
		settingCache:    mgmt.MgmtFactory.Management().V1().Setting().Cache(),
		nodeCache:       mgmt.CoreFactory.Core().V1().Node().Cache(),
		deploymentCache: mgmt.AppsFactory.Apps().V1().Deployment().Cache(),
		helmChartCache:  mgmt.HelmFactory.Helm().V1().HelmChart().Cache(),
	}
}

func (v *validator) Create(_ *admission.Request, newObj runtime.Object) error {
	upgrade := newObj.(*mgmtv1.Upgrade)
	if upgrade.Spec.Version == "" {
		return werror.BadRequest("Version is required")
	}

	if ok, err := v.validateCanUpgradeVersion(upgrade); !ok && err == nil {
		return werror.InternalError(fmt.Sprintf("Cannot upgrade to newer version %s", upgrade.Spec.Version))
	} else if err != nil {
		return err
	}

	upgrades, err := v.upgradeCache.List(labels.Everything())
	if err != nil {
		return werror.InternalError(fmt.Sprintf("Failed to list upgrades: %v", err))
	}

	for _, u := range upgrades {
		if u.Status.State == condition.StateComplete || u.Status.State == condition.StateError {
			msg := fmt.Sprintf("Cannot process until previous upgrade %q is complete", upgrades[0].Name)
			return werror.StatusConflict(msg)
		}
	}

	if upgrade.Annotations != nil {
		if skipWebhook, ok := upgrade.Annotations[constant.AnnotationSkipWebhook]; ok &&
			strings.ToLower(skipWebhook) == "true" {
			return nil
		}
	}

	return v.checkUpgradeResources()
}

func (v *validator) Update(_ *admission.Request, oldObj runtime.Object, newObj runtime.Object) error {
	upgrade := oldObj.(*mgmtv1.Upgrade)
	newUpgrade := newObj.(*mgmtv1.Upgrade)

	if upgrade.Spec.Version != newUpgrade.Spec.Version {
		return werror.StatusConflict("Upgrade version cannot be changed")
	}

	return nil
}

func (v *validator) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{"upgrades"},
		Scope:      admissionregv1.ClusterScope,
		APIGroup:   mgmtv1.SchemeGroupVersion.Group,
		APIVersion: mgmtv1.SchemeGroupVersion.Version,
		ObjectType: &mgmtv1.Upgrade{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}
