package api

import (
	"context"

	"github.com/rancher/steve/pkg/server"

	"github.com/llmos-ai/llmos-controller/pkg/api/chat"
	"github.com/llmos-ai/llmos-controller/pkg/server/config"
)

type registerSchema func(mgmt *config.Management, server *server.Server) error

var registers = []registerSchema{
	chat.RegisterSchema,
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
