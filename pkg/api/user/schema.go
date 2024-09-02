package user

import (
	"net/http"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/steve/pkg/schema"
	"github.com/rancher/steve/pkg/server"
	"github.com/rancher/wrangler/v3/pkg/schemas"

	"github.com/llmos-ai/llmos-operator/pkg/auth"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	userSchemaID         = "management.llmos.ai.user"
	ActionSetIsActive    = "setIsActive"
	ActionChangePassword = "changePassword"
)

type SetIsActiveInput struct {
	IsActive bool `json:"isActive"`
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func RegisterSchema(mgmt *config.Management, server *server.Server) error {
	users := mgmt.MgmtFactory.Management().V1().User()
	h := Handler{
		userClient: users,
		userCache:  users.Cache(),
		middleware: auth.NewMiddleware(mgmt),
	}

	server.BaseSchemas.MustImportAndCustomize(SetIsActiveInput{}, nil)
	server.BaseSchemas.MustImportAndCustomize(ChangePasswordInput{}, nil)
	t := []schema.Template{
		{
			ID: userSchemaID,
			Customize: func(s *types.APISchema) {
				s.CollectionFormatter = CollectionFormatter
				s.CollectionActions = map[string]schemas.Action{
					ActionChangePassword: {
						Input: "changePasswordInput",
					},
				}
				s.ListHandler = h.userListHandler
				s.Formatter = Formatter
				s.ResourceActions = map[string]schemas.Action{
					ActionSetIsActive: {
						Input: "setIsActiveInput",
					},
				}
				s.ActionHandlers = map[string]http.Handler{
					ActionSetIsActive:    h,
					ActionChangePassword: h,
				}
			},
		},
	}

	server.SchemaFactory.AddTemplate(t...)
	return nil
}
