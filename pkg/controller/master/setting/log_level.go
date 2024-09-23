package setting

import (
	"github.com/sirupsen/logrus"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
)

// setLogLevel updates the log level on setting changes
func (h *handler) setLogLevel(setting *mgmtv1.Setting) error {
	value := setting.Value
	if value == "" {
		value = setting.Default
	}
	level, err := logrus.ParseLevel(value)
	if err != nil {
		return err
	}

	logrus.Infof("set log level to %s", level)
	logrus.SetLevel(level)
	return nil
}
