package api

import (
	"context"

	"github.com/rancher/steve/pkg/server"

	"github.com/llmos-ai/llmos-operator/pkg/api/chat"
	"github.com/llmos-ai/llmos-operator/pkg/api/datasetversion"
	"github.com/llmos-ai/llmos-operator/pkg/api/model"
	"github.com/llmos-ai/llmos-operator/pkg/api/token"
	"github.com/llmos-ai/llmos-operator/pkg/api/user"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

type registerSchema func(scaled *config.Scaled, server *server.Server) error

var registers = []registerSchema{
	chat.RegisterSchema,
	user.RegisterSchema,
	token.RegisterSchema,
	model.RegisterSchema,
	datasetversion.RegisterSchema,
}

func registerSchemas(scaled *config.Scaled, server *server.Server, registers ...registerSchema) error {
	for _, register := range registers {
		if err := register(scaled, server); err != nil {
			return err
		}
	}
	return nil
}

func Register(ctx context.Context, server *server.Server) error {
	scaled := config.ScaledWithContext(ctx)
	return registerSchemas(scaled, server, registers...)
}
