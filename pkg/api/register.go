package api

import (
	"context"

	"github.com/rancher/steve/pkg/server"

	"github.com/llmos-ai/llmos-operator/pkg/api/chat"
	"github.com/llmos-ai/llmos-operator/pkg/api/token"
	"github.com/llmos-ai/llmos-operator/pkg/api/user"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

type registerSchema func(mgmt *config.Management, server *server.Server) error

var registers = []registerSchema{
	chat.RegisterSchema,
	user.RegisterSchema,
	token.RegisterSchema,
}

func registerSchemas(mgmt *config.Management, server *server.Server, registers ...registerSchema) error {
	for _, register := range registers {
		if err := register(mgmt, server); err != nil {
			return err
		}
	}
	return nil
}

func Register(_ context.Context, mgmt *config.Management, server *server.Server) error {
	return registerSchemas(mgmt, server, registers...)
}
