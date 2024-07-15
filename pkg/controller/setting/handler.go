package setting

import (
	"github.com/sirupsen/logrus"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
)

func (h *handler) settingOnChang(_ string, setting *mgmtv1.Setting) (*mgmtv1.Setting, error) {
	if setting == nil || setting.DeletionTimestamp != nil {
		return nil, nil
	}
	logrus.Debugf("setting on change, %+v", setting)

	toUpdate := setting.DeepCopy()
	if toUpdate.Annotations == nil {
		toUpdate.Annotations = make(map[string]string)
	}

	var err error
	if syncer, ok := syncers[setting.Name]; ok {
		err = syncer(setting)
		if err == nil {
			toUpdate.Annotations[constant.SettingPreConfiguredValueAnno] = setting.Value
		}
		if updateErr := h.setConfiguredCondition(toUpdate, err); updateErr != nil {
			return setting, updateErr
		}
	}
	return setting, nil
}

func (h *handler) setConfiguredCondition(settingCopy *mgmtv1.Setting, err error) error {
	if err != nil && (!mgmtv1.SettingConfigured.IsFalse(settingCopy) ||
		mgmtv1.SettingConfigured.GetMessage(settingCopy) != err.Error()) {
		mgmtv1.SettingConfigured.False(settingCopy)
		mgmtv1.SettingConfigured.Message(settingCopy, err.Error())
		if _, err := h.settings.Update(settingCopy); err != nil {
			return err
		}
	} else if err == nil {
		if settingCopy.Value == "" {
			mgmtv1.SettingConfigured.False(settingCopy)
		} else {
			mgmtv1.SettingConfigured.True(settingCopy)
		}
		mgmtv1.SettingConfigured.Message(settingCopy, "")
		if _, err := h.settings.Update(settingCopy); err != nil {
			return err
		}
	}
	return nil
}
