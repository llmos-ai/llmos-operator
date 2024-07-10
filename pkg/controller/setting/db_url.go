package setting

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	mgmtv1 "github.com/llmos-ai/llmos-controller/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-controller/pkg/constant"
	"github.com/llmos-ai/llmos-controller/pkg/database"
)

const postgresPrefix = "postgresql://"

func (h *handler) setDBUrl(setting *mgmtv1.Setting) error {
	url, err := h.getFormatedDBUrl(setting)
	if err != nil {
		return fmt.Errorf("failed to get formated DB url: %v", err)
	}

	logrus.Debugf("Run DB migration with url: %s", url)
	client, err := database.RunAutoMigrate(h.ctx, url)
	if err != nil {
		logrus.Debugf("failed to run DB auto migrate: %v", err)
		return err
	}
	// set ent client if DB url is updated
	h.mgmt.SetEntClient(client)
	return nil
}

func (h *handler) getFormatedDBUrl(setting *mgmtv1.Setting) (string, error) {
	baseUrl := setting.Value
	if baseUrl == "" {
		baseUrl = setting.Default
	}
	baseUrl = strings.TrimPrefix(baseUrl, postgresPrefix)

	secretRefName := setting.Annotations[constant.SecretNameRefAnno]
	if secretRefName == "" {
		secretRefName = constant.DefaultDBSecretName
	}
	defaultSecret, err := h.secretCache.Get(constant.SystemNamespaceName, secretRefName)
	if err != nil {
		return "", err
	}
	username := defaultSecret.Data[constant.DBUsernameKey]
	password := defaultSecret.Data[constant.DBUserPasswordKey]
	dbName := defaultSecret.Data[constant.DBDatabaseKey]

	return fmt.Sprintf("postgresql://%s:%s@%s/%s",
		username,
		password,
		baseUrl,
		dbName,
	), nil
}
