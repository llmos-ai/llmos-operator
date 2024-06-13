package constant

import (
	"time"
)

const (
	LLMOSPrefix       = "llmos.ai"
	LLMOSVersionLabel = LLMOSPrefix + "/version"
	LLMOSUpgradeLabel = LLMOSPrefix + "/upgrade"
	LLMOSManagedLabel = LLMOSPrefix + "/managed"

	SettingPreConfiguredValueAnno = LLMOSPrefix + "/previous-configured-value"
	SecretNameRefAnno             = LLMOSPrefix + "/secret-name"

	// Secret reference keys of the PostgreSQL, these are defined in the Helm chart values
	DefaultDBSecretName      = "llmos-db-credentials"
	DBUsernameKey            = "pg-username"
	DBDatabaseKey            = "pg-database"
	DBUserPasswordKey        = "pg-password"
	DBAdminPasswordKey       = "pg-admin-password"
	DBReplicationPasswordKey = "pg-replica-password"

	ModelOriginModelAnnotation    = LLMOSPrefix + "/original-model"
	ModelFileSkipDeleteAnnotation = LLMOSPrefix + "/model-file-skip-delete"

	LLMOSSystemNamespace = "llmos-system"
	LLMOSPublicNamespace = "llmos-public"
	SUCNamespace         = "system-upgrade"
	TimeLayout           = time.RFC3339

	AdminRole = "cluster-admin"
)
