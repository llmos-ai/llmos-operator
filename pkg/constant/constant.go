package constant

const (
	LLMOSPrefix                        = "llmos.ai"
	LLMOSVersionLabel                  = LLMOSPrefix + "/version"
	LLMOSUpgradeLabel                  = LLMOSPrefix + "/upgrade"
	LLMOSManagedLabel                  = LLMOSPrefix + "/managed"
	AnnotationClusterPolicyProviderKey = LLMOSPrefix + "k8sProvider"

	LLMOSSystemNamespace = "llmos-system"
	SUCNamespace         = "system-upgrade"
)
