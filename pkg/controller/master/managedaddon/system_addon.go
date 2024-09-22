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
)

// systemAddonTemplates are defined in the pkg/templates/addons
var systemAddonTemplates = []string{
	"gpu-operator.yaml",
	"kuberay-operator.yaml",
	"rook-ceph.yaml",
	"llmos-operator-redis.yaml",
}

func (h *handler) registerSystemAddons(_ context.Context) error {
	serverVersion := settings.ServerVersion.Get()
	for _, fileName := range systemAddonTemplates {
		template, err := template.Render(template.AddonTemplate, fileName, nil)
		if err != nil {
			return fmt.Errorf("failed to render template: %w", err)
		}

		if len(template.Bytes()) == 0 {
			logrus.Warnf("template is empty: %s", fileName)
			continue
		}

		addonTemplate := &mgmtv1.ManagedAddon{}
		err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(template.Bytes()), 1024).Decode(addonTemplate)
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

		if _, err = h.createOrUpdateAddon(addonTemplate, serverVersion); err != nil {
			return err
		}
	}

	return nil
}

func (h *handler) createOrUpdateAddon(addonTemplate *mgmtv1.ManagedAddon,
	serverVersion string) (*mgmtv1.ManagedAddon, error) {
	addon, err := h.managedAddon.Get(addonTemplate.Namespace, addonTemplate.Name, metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		return h.managedAddons.Create(addonTemplate)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get addon %s/%s, error: %s",
			addonTemplate.Namespace, addonTemplate.Name, err.Error())
	}

	logrus.Tracef("addon %s/%s already exists, %+v", addon.Namespace, addon.Name, addon)
	if !reflect.DeepEqual(addon.Spec, addonTemplate.Spec) ||
		addon.Labels[constant.LLMOSServerVersionLabel] != serverVersion {
		addonCpy := addon.DeepCopy()
		addonCpy.Spec = addonTemplate.Spec
		addonCpy.Labels[constant.LLMOSServerVersionLabel] = serverVersion
		logrus.Debugf("addon %s/%s spec is not equal, update it", addonCpy.Name, addonCpy.Namespace)
		return h.managedAddons.Update(addonCpy)
	}

	return nil, nil
}
