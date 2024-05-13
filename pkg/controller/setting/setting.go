package setting

import (
	"context"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mgmtv1 "github.com/llmos-ai/llmos-controller/pkg/apis/management.llmos.ai/v1"
	ctlmgmtv1 "github.com/llmos-ai/llmos-controller/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-controller/pkg/server/config"
	"github.com/llmos-ai/llmos-controller/pkg/settings"
)

type handler struct {
	settings     ctlmgmtv1.SettingClient
	settingCache ctlmgmtv1.SettingCache
	fallback     map[string]string
}

func Register(_ context.Context, mgmt *config.Management) error {
	sp := &handler{
		settings:     mgmt.LLMOSMgmtFactory.Management().V1().Setting(),
		settingCache: mgmt.LLMOSMgmtFactory.Management().V1().Setting().Cache(),
		fallback:     map[string]string{},
	}

	return settings.SetProvider(sp)
}

func (h *handler) Get(name string) string {
	value := os.Getenv(settings.GetEnvKey(name))
	if value != "" {
		return value
	}
	obj, err := h.settingCache.Get(name)
	if err != nil {
		val, err := h.settings.Get(name, v1.GetOptions{})
		if err != nil {
			return h.fallback[name]
		}
		obj = val
	}
	if obj.Value == "" {
		return obj.Default
	}
	return obj.Value
}

func (h *handler) Set(name, value string) error {
	envValue := os.Getenv(settings.GetEnvKey(name))
	if envValue != "" {
		return fmt.Errorf("setting %s can not be set because it is from environment variable", name)
	}
	obj, err := h.settings.Get(name, v1.GetOptions{})
	if err != nil {
		return err
	}

	obj.Value = value
	_, err = h.settings.Update(obj)
	return err
}

func (h *handler) SetIfUnset(name, value string) error {
	obj, err := h.settings.Get(name, v1.GetOptions{})
	if err != nil {
		return err
	}

	if obj.Value != "" {
		return nil
	}

	obj.Value = value
	_, err = h.settings.Update(obj)
	return err
}

func (h *handler) SetAll(settingsMap map[string]settings.Setting) error {
	fallback := map[string]string{}

	for name, setting := range settingsMap {
		key := settings.GetEnvKey(name)
		value := os.Getenv(key)

		obj, err := h.settings.Get(setting.Name, v1.GetOptions{})
		if errors.IsNotFound(err) {
			newSetting := &mgmtv1.Setting{
				ObjectMeta: v1.ObjectMeta{
					Name: setting.Name,
				},
				Default: setting.Default,
			}
			if value != "" {
				newSetting.Value = value
			}
			if newSetting.Value == "" {
				fallback[newSetting.Name] = newSetting.Default
			} else {
				fallback[newSetting.Name] = newSetting.Value
			}
			_, err := h.settings.Create(newSetting)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			update := false
			if obj.Default != setting.Default {
				obj.Default = setting.Default
				update = true
			}
			if value != "" && obj.Value != value {
				obj.Value = value
				update = true
			}
			if obj.Value == "" {
				fallback[obj.Name] = obj.Default
			} else {
				fallback[obj.Name] = obj.Value
			}
			if update {
				_, err := h.settings.Update(obj)
				if err != nil {
					return err
				}
			}
		}
	}

	h.fallback = fallback

	return nil
}
