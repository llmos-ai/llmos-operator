package knowledgebase

import (
	"net/http"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/steve/pkg/schema"
	"github.com/rancher/steve/pkg/server"
	"github.com/rancher/wrangler/v3/pkg/schemas"

	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const knowledgebaseSchemaID = "agent.llmos.ai.knowledgebase"

func Formatter(request *types.APIRequest, resource *types.RawResource) {
	resource.Actions = make(map[string]string, 1)
	resource.AddAction(request, ActionSearch)
	resource.AddAction(request, ActionListObjects)
}

func RegisterSchema(scaled *config.Scaled, server *server.Server) error {
	h := NewHandler(scaled)

	server.BaseSchemas.MustImportAndCustomize(SearchInput{}, nil)
	server.BaseSchemas.MustImportAndCustomize(ListObjectsInput{}, nil)

	customizeFunc := func(s *types.APISchema) {
		s.Formatter = Formatter
		s.ResourceActions = map[string]schemas.Action{
			ActionSearch: {
				Input: "searchInput",
			},
			ActionListObjects: {
				Input: "listObjectsInput",
			},
		}
		s.ActionHandlers = map[string]http.Handler{
			ActionSearch:      h,
			ActionListObjects: h,
		}
	}

	t := []schema.Template{
		{
			ID:        knowledgebaseSchemaID,
			Customize: customizeFunc,
		},
	}

	server.SchemaFactory.AddTemplate(t...)
	return nil
}
