package constant

import (
	"time"
)

const (
	LLMOSPrefix       = "llmos.ai"
	LLMOSVersionLabel = LLMOSPrefix + "/version"
	LLMOSUpgradeLabel = LLMOSPrefix + "/upgrade"
	LLMOSManagedLabel = LLMOSPrefix + "/managed"

	ModelOriginModelAnnotation    = LLMOSPrefix + "/original-model"
	ModelFileSkipDeleteAnnotation = LLMOSPrefix + "/model-file-skip-delete"

	LLMOSSystemNamespace = "llmos-system"
	LLMOSPublicNamespace = "llmos-public"
	SUCNamespace         = "system-upgrade"
	TimeLayout           = time.RFC3339
)
