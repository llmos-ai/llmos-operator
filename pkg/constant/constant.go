package constant

import (
	"time"
)

const (
	LLMOSPrefix = "llmos.ai"
	MLPrefix    = "ml.llmos.ai"
	MgmtPrefix  = "management.llmos.ai"

	SystemNamespaceName     = "llmos-system"
	CephSystemNamespaceName = "ceph-system"
	PublicNamespaceName     = "llmos-public"
	SUCNamespace            = "system-upgrade"
	CephClusterName         = "llmos-ceph"
	AdminRole               = "cluster-admin"

	TimeLayout = time.RFC3339

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

	AnnotationResourceStopped          = LLMOSPrefix + "/resource-stopped"
	AnnotationVolumeClaimTemplates     = LLMOSPrefix + "/volume-claim-templates"
	AnnotationClusterPolicyProviderKey = LLMOSPrefix + "/k8s-provider"

	AnnotationAddRookCephRbac           = LLMOSPrefix + "/add-ceph-cluster-rbac"
	AnnotationAddedRookCephRbac         = LLMOSPrefix + "/added-ceph-cluster-rbac"
	AnnotationAddRookCephBlockStorage   = LLMOSPrefix + "/add-ceph-cluster-block-storage"
	AnnotationAddedRookCephBlockStorage = LLMOSPrefix + "/added-ceph-cluster-block-storage"
	AnnotationAddRookCephFilesystem     = LLMOSPrefix + "/add-ceph-cluster-filesystem"
	AnnotationAddedRookCephFilesystem   = LLMOSPrefix + "/added-ceph-cluster-filesystem"
	AnnotationAddCephToolbox            = LLMOSPrefix + "/add-ceph-toolbox"
	AnnotationAddedCephToolbox          = LLMOSPrefix + "/added-ceph-toolbox"

	// kubeRay related constant
	LabelRaySchedulerName           = "ray.io/scheduler-name"
	AnnotationRayClusterInitialized = MLPrefix + "rayClusterInitialized"
	AnnotationRayFTEnabledKey       = "ray.io/ft-enabled"
	RayRedisCleanUpFinalizer        = "ray.io/gcs-ft-redis-cleanup-finalizer"

	RayServiceKind     = "RayService"
	RedisSecretKeyName = "redis-password" // #nosec G101

	DefaultAdminLabelKey = MgmtPrefix + "/default-admin"
)
