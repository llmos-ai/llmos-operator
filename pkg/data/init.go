package data

import (
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

// Init adds built-in resources
func Init(mgmt *config.Management) error {
	if err := addDefaultNamespaces(mgmt.Apply); err != nil {
		return err
	}

	// bootstrap global roles first before adding any user
	if err := BootstrapGlobalRoles(mgmt); err != nil {
		return err
	}

	return BootstrapDefaultAdmin(mgmt)
}
