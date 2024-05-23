package data

import (
	"github.com/llmos-ai/llmos-controller/pkg/server/config"
)

// Init adds built-in resources
func Init(mgmt *config.Management) error {
	if err := addPublicNamespace(mgmt.Apply); err != nil {
		return err
	}

	return nil
}
