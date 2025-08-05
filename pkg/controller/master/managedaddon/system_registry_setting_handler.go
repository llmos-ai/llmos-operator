package managedaddon

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
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

var (
	repositoryKeys     = []string{"registry", "repository", "image_registry"}
	repositoryPrefixes = []string{"ghcr.io", "nvcr.io", "quay.io", "docker.io"}
)

type SettingHandler struct {
	settings     ctlmgmtv1.SettingClient
	settingCache ctlmgmtv1.SettingCache
	addons       ctlmgmtv1.ManagedAddonClient
	addonCache   ctlmgmtv1.ManagedAddonCache

	AddonHandler *AddonHandler
}

func (s *SettingHandler) systemRegistryOnChange(_ string, setting *mgmtv1.Setting) (*mgmtv1.Setting, error) {
	if setting == nil || setting.DeletionTimestamp != nil {
		return setting, nil
	}

	if setting.Name != settings.GlobalSystemImageRegistryName ||
		(settings.GlobalSystemImageRegistry.Get() == "" && setting.Value == "") {
		return setting, nil
	}

	// Re-sync system addon
	err := s.AddonHandler.registerSystemAddons()
	if err != nil {
		return setting, err
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
	if registry == "" {
		return yamlString, nil
	}
	var config map[string]interface{}
	err := yaml.Unmarshal([]byte(yamlString), &config)
	if err != nil {
		return "", fmt.Errorf("error parsing managed addon values: %v", err)
	}

	if config == nil {
		config = make(map[string]interface{})
	}

	updateValues(config, registry)

	modifiedYAML, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("error converting managed addon values to YAML: %v", err)
	}

	return string(modifiedYAML), nil
}

func updateValues(data map[string]interface{}, registry string) {
	// for global image registry setting
	if _, ok := data[globalKey]; ok {
		if global, ok := data[globalKey].(map[string]interface{}); ok {
			global[imageRegistryKey] = registry
		}
	} else {
		data[globalKey] = map[string]interface{}{
			imageRegistryKey: registry,
		}
	}

	traverseAndUpdate(data, registry)
}

func traverseAndUpdate(data interface{}, registry string) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			if contains(repositoryKeys, key) {
				if repoStr, ok := val.(string); ok {
					if hasPrefix(repoStr, repositoryPrefixes) {
						logrus.Debugf("update %s with repository: %s", key, repoStr)
						parts := strings.SplitN(repoStr, "/", 2)
						if len(parts) > 1 {
							v[key] = fmt.Sprintf("%s/%s", registry, parts[1])
						} else {
							v[key] = registry
						}
					}
				}
			} else {
				traverseAndUpdate(val, registry)
			}
		}
	case []interface{}:
		for _, item := range v {
			traverseAndUpdate(item, registry)
		}
	}
}

// contains checks if a string exists in a slice of strings
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func hasPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
