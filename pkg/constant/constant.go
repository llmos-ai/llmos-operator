package constant

import (
	"time"
)

const (
	LLMOSPrefix = "llmos.ai"
	MLPrefix    = "ml.llmos.ai"
	MgmtPrefix  = "management.llmos.ai"

	SystemNamespaceName        = "llmos-system"
	StorageSystemNamespaceName = "storage-system"
	PublicNamespaceName        = "llmos-public"
	SUCNamespace               = "system-upgrade"
	CephClusterName            = "llmos-ceph"
	AdminRole                  = "cluster-admin"
	NvidiaGPUKey               = "nvidia.com/gpu"
	KubeSystemNamespaceName    = "kube-system"
	MonitoringNamespace        = "llmos-monitoring-system"

	LLMOSCrdChartName      = "llmos-crd"
	LLMOSOperatorChartName = "llmos-operator"

	TrueStr    = "true"
	TimeLayout = time.RFC3339

	KubeNodeRoleLabelPrefix      = "node-role.kubernetes.io/"
	KubeMasterNodeLabelKey       = KubeNodeRoleLabelPrefix + "master"
	KubeControlPlaneNodeLabelKey = KubeNodeRoleLabelPrefix + "control-plane"
	KubeEtcdNodeLabelKey         = KubeNodeRoleLabelPrefix + "etcd"
	KubeWorkerNodeLabelKey       = KubeNodeRoleLabelPrefix + "worker"

	LabelEnforcePss = "pod-security.kubernetes.io/enforce"
	LabelAppName    = "app.kubernetes.io/name"
	LabelAppVersion = "app.kubernetes.io/version"

	LLMOSVersionLabel         = LLMOSPrefix + "/version"
	LLMOSServerVersionLabel   = LLMOSPrefix + "/server-version"
	LLMOSManagedLabel         = LLMOSPrefix + "/managed"
	ManagedAddonLabel         = LLMOSPrefix + "/managed-addon"
	SystemAddonLabel          = LLMOSPrefix + "/system-addon"
	SystemAddonAllowEditLabel = LLMOSPrefix + "/system-addon-allow-edit"

	SettingPreConfiguredValueAnno = LLMOSPrefix + "/previous-configured-value"
	SecretNameRefAnno             = LLMOSPrefix + "/secret-name"

	// Secret reference keys of the PostgreSQL, these are defined in the Helm chart values
	DefaultDBSecretName      = "llmos-db-credentials"
	DBUsernameKey            = "pg-username"
	DBDatabaseKey            = "pg-database"
	DBUserPasswordKey        = "pg-password"
	DBAdminPasswordKey       = "pg-admin-password"
	DBReplicationPasswordKey = "pg-replica-password"

	AnnotationResourceStopped          = LLMOSPrefix + "/resource-stopped"
	AnnotationVolumeClaimTemplates     = LLMOSPrefix + "/volume-claim-templates"
	AnnotationClusterPolicyProviderKey = LLMOSPrefix + "/k8s-provider"
	AnnotationSkipWebhook              = LLMOSPrefix + "/skip-webhook"

	/*
		KubeRay related constant
	*/
	LabelRaySchedulerName           = "ray.io/scheduler-name"
	AnnotationRayClusterInitialized = MLPrefix + "rayClusterInitialized"
	AnnotationRayFTEnabledKey       = "ray.io/ft-enabled"
	RayRedisCleanUpFinalizer        = "ray.io/gcs-ft-redis-cleanup-finalizer"

	RayServiceKind     = "RayService"
	RedisSecretKeyName = "redis-password" // #nosec G101

	/*
		Management related constants
	*/
	DefaultAdminLabelKey       = MgmtPrefix + "/default-admin"
	LabelManagementUsernameKey = MgmtPrefix + "/username"
	LabelManagementUserIdKey   = MgmtPrefix + "/user-id"

	/*
		ML related constants
	*/
	LabelNotebookName            = MLPrefix + "/notebook-name"
	LabelLLMOSMLAppName          = MLPrefix + "/app"
	LabelLLMOSMLType             = MLPrefix + "/type"
	LabelModelServiceName        = MLPrefix + "/model-service-name"
	LabelModelServiceServeEngine = MLPrefix + "/serve-engine"
)
