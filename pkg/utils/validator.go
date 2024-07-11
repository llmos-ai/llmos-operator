package utils

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func ValidateVolumeClaimTemplatesAnnotation(pvcAnno string) error {
	var pvcs []*corev1.PersistentVolumeClaim
	if err := json.Unmarshal([]byte(pvcAnno), &pvcs); err != nil {
		return err
	}
	for _, pvc := range pvcs {
		if pvc.Name == "" {
			return fmt.Errorf("PVC name is required")
		}
	}
	return nil
}
