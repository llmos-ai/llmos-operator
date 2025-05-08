package common

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"

	ctlstoragev1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/storage.k8s.io/v1"
)

const storageClassName = "llmos-ceph-block"

func CheckStorageClassExists(scCache ctlstoragev1.StorageClassCache) error {
	if _, err := scCache.Get(storageClassName); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("storage class %s not found, please enable system storage firstly", storageClassName)
		}
		return fmt.Errorf("failed to get storage class %s: %w", storageClassName, err)
	}
	return nil
}
