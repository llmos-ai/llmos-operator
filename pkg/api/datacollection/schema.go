package datacollection

import (
	"net/http"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/steve/pkg/schema"
	"github.com/rancher/steve/pkg/server"
	"github.com/rancher/wrangler/v3/pkg/schemas"

	cr "github.com/llmos-ai/llmos-operator/pkg/api/common/registry"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const applicationDataSchemaID = "agent.llmos.ai.datacollection"

func Formatter(request *types.APIRequest, resource *types.RawResource) {
	resource.Actions = make(map[string]string, 1)
	if request.AccessControl.CanUpdate(request, resource.APIObject, resource.Schema) != nil {
		return
	}
	resource.AddAction(request, cr.ActionUpload)
	resource.AddAction(request, cr.ActionList)
	resource.AddAction(request, cr.ActionRemove)
	resource.AddAction(request, cr.ActionGeneratePresignedURL)
}

func RegisterSchema(scaled *config.Scaled, server *server.Server) error {
	h := NewHandler(scaled)

	server.BaseSchemas.MustImportAndCustomize(cr.UploadInput{}, nil)
	server.BaseSchemas.MustImportAndCustomize(cr.ListInput{}, nil)
	server.BaseSchemas.MustImportAndCustomize(cr.RemoveInput{}, nil)
	server.BaseSchemas.MustImportAndCustomize(cr.GeneratePresignedURLInput{}, nil)

	customizeFunc := func(s *types.APISchema) {
		s.Formatter = Formatter
		s.ResourceActions = map[string]schemas.Action{
			cr.ActionUpload: {
				Input: "uploadInput",
			},
			cr.ActionList: {
				Input: "listInput",
			},
			cr.ActionRemove: {
				Input: "removeInput",
			},
			cr.ActionGeneratePresignedURL: {
				Input: "generatePresignedURLInput",
			},
		}
		s.ActionHandlers = map[string]http.Handler{
			cr.ActionUpload:               h,
			cr.ActionList:                 h,
			cr.ActionRemove:               h,
			cr.ActionGeneratePresignedURL: h,
		}
	}

	t := []schema.Template{
		{
			ID:        applicationDataSchemaID,
			Customize: customizeFunc,
		},
	}

	server.SchemaFactory.AddTemplate(t...)
	return nil
}
