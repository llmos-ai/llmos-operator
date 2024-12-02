package setting

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
)

type ManagedAddonConfigs struct {
	AddonConfigs []AddonConfig `json:",inline"`
}

type AddonConfig struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

func (h *handler) setManagedAddonConfigs(setting *mgmtv1.Setting) error {
	val, err := h.constructedManagedAddonConfigs()
	if err != nil {
		return err
	}
	return h.Set(setting.Name, val)
}

func (h *handler) constructedManagedAddonConfigs() (string, error) {
	addons, err := h.managedAddonCache.List(metav1.NamespaceAll, labels.SelectorFromSet(map[string]string{
		"llmos.ai/cluster-tools": "true",
	}))
	if err != nil {
		return "", err
	}

	cfg := ManagedAddonConfigs{}

	for _, addon := range addons {
		cfg.AddonConfigs = append(cfg.AddonConfigs, AddonConfig{
			Name:    addon.Name,
			Enabled: addon.Spec.Enabled,
			Status:  string(addon.Status.State),
		})
	}

	cfgStr, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}

	return string(cfgStr), nil
}

func DecodeManagedAddonConfigs(val string) (*ManagedAddonConfigs, error) {
	cfg := &ManagedAddonConfigs{}
	err := json.Unmarshal([]byte(val), cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
