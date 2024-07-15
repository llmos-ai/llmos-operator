package chat

import (
	"net/http"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/steve/pkg/server"

	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const chatTypeName = "chat"

func RegisterSchema(mgmt *config.Management, server *server.Server) error {
	schemas := server.BaseSchemas
	schemas.InternalSchemas.TypeName(chatTypeName, Chat{})
	// import the struct EjectCdRomActionInput to the schema, then the action could use it as input,
	// and because wrangler converts the struct typeName to lower title, so the action input should start with lower case.
	// https://github.com/rancher/wrangler/blob/master/pkg/schemas/reflection.go#L26
	schemas.MustImportAndCustomize(NewChatRequest{}, nil)
	schemas.MustImportAndCustomize(UpdateChatRequest{}, nil)
	schemas.MustImportAndCustomize(Chat{}, func(schema *types.APISchema) {
		schema.CollectionMethods = []string{http.MethodGet, http.MethodPost}
		schema.ResourceMethods = []string{http.MethodGet, http.MethodPut, http.MethodDelete}
		schema.Store = &Store{
			handler: NewHandler(mgmt),
		}
	})
	return nil
}
