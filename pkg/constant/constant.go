package constant

import (
	"time"
)

const (
	LLMOSPrefix = "llmos.ai"
	MLPrefix    = "ml.llmos.ai"

	SystemNamespaceName = "llmos-system"
	PublicNamespaceName = "llmos-public"
	SUCNamespace        = "system-upgrade"
	AdminRole           = "cluster-admin"
	TimeLayout          = time.RFC3339

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

	AnnotationResourceStopped          = LLMOSPrefix + "/resourceStopped"
	AnnotationVolumeClaimTemplates     = LLMOSPrefix + "/volumeClaimTemplates"
	AnnotationClusterPolicyProviderKey = LLMOSPrefix + "/k8sProvider"

	// kubeRay related constant
	LabelRaySchedulerName           = "ray.io/scheduler-name"
	AnnotationRayClusterInitialized = MLPrefix + "rayClusterInitialized"
	AnnotationRayFTEnabledKey       = "ray.io/ft-enabled"
	RayRedisCleanUpFinalizer        = "ray.io/gcs-ft-redis-cleanup-finalizer"
	RayServiceKind                  = "RayService"
	RedisSecretKeyName              = "redis-password" // #nosec G101

	DefaultAdminLabelKey = "management.llmos.ai/default-admin"
)
