package managedaddon

import (
	"fmt"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/labels"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
)

const (
	globalKey        = "global"
	imageRegistryKey = "imageRegistry"
)

type SettingHandler struct {
	settings     ctlmgmtv1.SettingClient
	settingCache ctlmgmtv1.SettingCache
	addons       ctlmgmtv1.ManagedAddonClient
	addonCache   ctlmgmtv1.ManagedAddonCache
}

func (s *SettingHandler) systemRegistryOnChange(_ string, setting *mgmtv1.Setting) (*mgmtv1.Setting, error) {
	if setting == nil || setting.DeletionTimestamp != nil {
		return setting, nil
	}

	if setting.Name != settings.GlobalSystemImageRegistryName ||
		(settings.GlobalSystemImageRegistry.Get() == "" && setting.Value == "") {
		return setting, nil
	}

	// Fetch all system addons and update default global registry value
	selector := labels.SelectorFromSet(map[string]string{
		constant.SystemAddonLabel: "true",
	})

	addons, err := s.addonCache.List("", selector)
	if err != nil {
		return setting, err
	}

	for _, addon := range addons {
		values := addon.Spec.ValuesContent
		newValues, err := ModifyImageRegistry(values, settings.GlobalSystemImageRegistry.Get())
		if err != nil {
			return setting, err
		}
		if newValues != values {
			toUpdate := addon.DeepCopy()
			toUpdate.Spec.ValuesContent = newValues
			if _, err := s.addons.Update(toUpdate); err != nil {
				return setting, err
			}
		}
	}

	return setting, nil
}

func ModifyImageRegistry(yamlString string, registry string) (string, error) {
	// Parse the YAML into a map
	var config map[string]interface{}
	err := yaml.Unmarshal([]byte(yamlString), &config)
	if err != nil {
		return "", fmt.Errorf("error parsing managed addon values: %v", err)
	}

	if config == nil {
		config = make(map[string]interface{})
	}

	// Navigate and modify the value
	if global, ok := config[globalKey].(map[string]interface{}); ok {
		global[imageRegistryKey] = registry
	} else {
		// Add 'global' if not present
		config[globalKey] = map[string]interface{}{
			imageRegistryKey: registry,
		}
	}

	// Convert the map back to a YAML string
	modifiedYAML, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("error converting managed addon values to YAML: %v", err)
	}

	return string(modifiedYAML), nil
}
