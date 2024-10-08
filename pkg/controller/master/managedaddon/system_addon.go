package managedaddon

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
	"github.com/llmos-ai/llmos-operator/pkg/template"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

func (h *handler) registerSystemAddons(_ context.Context) error {
	serverVersion := settings.ServerVersion.Get()

	systemAddonTemplates, err := template.GetAllFilenames()
	if err != nil {
		return fmt.Errorf("failed to get all addon templates: %w", err)
	}

	for _, fileName := range systemAddonTemplates {
		templateFile, err := template.Render(template.AddonTemplate, fileName, nil)
		if err != nil {
			return fmt.Errorf("failed to render template: %w", err)
		}

		if len(templateFile.Bytes()) == 0 {
			logrus.Warnf("template is empty: %s", fileName)
			continue
		}

		addonTemplate := &mgmtv1.ManagedAddon{}
		err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(templateFile.Bytes()), 1024).Decode(addonTemplate)
		if err != nil {
			return fmt.Errorf("failed to decode %s, error: %s", fileName, err.Error())
		}

		if !strings.Contains(fileName, addonTemplate.Name) {
			return fmt.Errorf("addon name %s is not equal to file name %s", addonTemplate.Name, fileName)
		}

		if addonTemplate.Labels == nil {
			addonTemplate.Labels = make(map[string]string)
		}
		addonTemplate.Labels[constant.LLMOSServerVersionLabel] = serverVersion
		if _, err = h.reconcileManagedAddon(addonTemplate, serverVersion); err != nil {
			return err
		}
	}

	return nil
}

func (h *handler) reconcileManagedAddon(addonTemplate *mgmtv1.ManagedAddon, serverVersion string) (
	*mgmtv1.ManagedAddon, error) {
	addon, err := h.managedAddon.Get(addonTemplate.Namespace, addonTemplate.Name, metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		return h.managedAddons.Create(addonTemplate)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get addon %s/%s, error: %s",
			addonTemplate.Namespace, addonTemplate.Name, err.Error())
	}

	// Skip updating addon if it is being deleted
	if addon == nil || addon.DeletionTimestamp != nil {
		return addonTemplate, nil
	}

	toUpdate := addon.DeepCopy()
	toUpdate.Spec.Version = addonTemplate.Spec.Version
	toUpdate.Spec.Chart = addonTemplate.Spec.Chart
	toUpdate.Spec.Repo = addonTemplate.Spec.Repo
	toUpdate.Spec.DefaultValuesContent = addonTemplate.Spec.DefaultValuesContent
	toUpdate.Spec.FailurePolicy = addonTemplate.Spec.FailurePolicy

	// Merge labels and annotations
	if toUpdate.Labels == nil {
		toUpdate.Labels = make(map[string]string)
	}
	if toUpdate.Annotations == nil {
		toUpdate.Annotations = make(map[string]string)
	}
	toUpdate.Labels = utils.MergeMapString(toUpdate.Labels, addonTemplate.Labels)
	toUpdate.Annotations = utils.MergeMapString(toUpdate.Annotations, addonTemplate.Annotations)
	toUpdate.Labels[constant.LLMOSServerVersionLabel] = serverVersion

	if !reflect.DeepEqual(addon, toUpdate) {
		logrus.Debugf("system addon %s/%s has changed, updating it", addon.Name, addon.Namespace)
		return h.managedAddons.Update(toUpdate)
	}

	return toUpdate, nil
}
