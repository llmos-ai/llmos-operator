package registry

import (
	"net/http"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/steve/pkg/schema"
	"github.com/rancher/steve/pkg/server"
	"github.com/rancher/wrangler/v3/pkg/schemas"

	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	modelVersionSchemaID   = "ml.llmos.ai.modelversion"
	datasetVersionSchemaID = "ml.llmos.ai.datasetversion"

	ActionUpload          = "upload"
	ActionDownload        = "download"
	ActionList            = "list"
	ActionRemove          = "remove"
	ActionCreateDirectory = "createDirectory"
)

type UploadInput struct {
	SourceFilePath string `json:"sourceFilePath"`
	// if empty, use version as target directory
	TargetDirectory string `json:"targetDirectory"`
}
type DownloadInput struct {
	TargetFilePath string `json:"targetFilePath"`
}
type ListInput DownloadInput
type RemoveInput DownloadInput

type CreateDirectoryInput struct {
	TargetDirectory string `json:"targetDirectory"`
}

func RegisterSchema(scaled *config.Scaled, server *server.Server) error {
	h := NewHandler(scaled)

	server.BaseSchemas.MustImportAndCustomize(UploadInput{}, nil)
	server.BaseSchemas.MustImportAndCustomize(DownloadInput{}, nil)
	server.BaseSchemas.MustImportAndCustomize(ListInput{}, nil)
	server.BaseSchemas.MustImportAndCustomize(RemoveInput{}, nil)
	server.BaseSchemas.MustImportAndCustomize(CreateDirectoryInput{}, nil)

	customizeFunc := func(s *types.APISchema) {
		s.Formatter = Formatter
		s.ResourceActions = map[string]schemas.Action{
			ActionUpload: {
				Input: "uploadInput",
			},
			ActionDownload: {
				Input: "downloadInput",
			},
			ActionList: {
				Input: "listInput",
			},
			ActionRemove: {
				Input: "removeInput",
			},
			ActionCreateDirectory: {
				Input: "createDirectoryInput",
			},
		}
		s.ActionHandlers = map[string]http.Handler{
			ActionUpload:          h,
			ActionDownload:        h,
			ActionList:            h,
			ActionRemove:          h,
			ActionCreateDirectory: h,
		}
	}

	t := []schema.Template{
		{
			ID:        modelVersionSchemaID,
			Customize: customizeFunc,
		},
		{
			ID:        datasetVersionSchemaID,
			Customize: customizeFunc,
		},
	}

	server.SchemaFactory.AddTemplate(t...)
	return nil
}
