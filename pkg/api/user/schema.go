package user

import (
	"net/http"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/steve/pkg/schema"
	"github.com/rancher/steve/pkg/server"
	"github.com/rancher/wrangler/v2/pkg/schemas"

	ctlmgmtv1 "github.com/llmos-ai/llmos-controller/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-controller/pkg/server/config"
)

const (
	userSchemaID      = "management.llmos.ai.user"
	ActionSetIsActive = "setIsActive"
)

type Handler struct {
	httpClient http.Client
	user       ctlmgmtv1.UserClient
	userCache  ctlmgmtv1.UserCache
}

type SetIsActiveInput struct {
	IsActive bool `json:"isActive"`
}

func RegisterSchema(mgmt *config.Management, server *server.Server) error {
	users := mgmt.MgmtFactory.Management().V1().User()
	h := Handler{
		httpClient: http.Client{},
		user:       users,
		userCache:  users.Cache(),
	}

	server.BaseSchemas.MustImportAndCustomize(SetIsActiveInput{}, nil)
	t := []schema.Template{
		{
			ID:        userSchemaID,
			Formatter: formatter,
			Customize: func(apiSchema *types.APISchema) {
				apiSchema.ListHandler = h.userListHandler
				apiSchema.ResourceActions = map[string]schemas.Action{
					ActionSetIsActive: {
						Input: "setIsActiveInput",
					},
				}
				apiSchema.ActionHandlers = map[string]http.Handler{
					ActionSetIsActive: &h,
				}
			},
		},
	}

	server.SchemaFactory.AddTemplate(t...)
	return nil
}
