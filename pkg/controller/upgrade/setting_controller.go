package upgrade

import (
	"github.com/sirupsen/logrus"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
)

// settingHandler do version syncs on server-version setting changes
type settingHandler struct {
	versionSyncer *versionSyncer
}

func (h *settingHandler) syncerOnChange(_ string, setting *mgmtv1.Setting) (*mgmtv1.Setting, error) {
	if setting == nil || setting.DeletionTimestamp != nil || setting.Name != settings.ServerVersionName {
		return setting, nil
	}

	if err := h.versionSyncer.sync(); err != nil {
		logrus.Errorf("failed to sync versions: %v", err)
	}

	return setting, nil
}
