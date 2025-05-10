package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/sirupsen/logrus"

	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/registry/backend"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

const (
	ActionUpload          = "upload"
	ActionDownload        = "download"
	ActionList            = "list"
	ActionRemove          = "remove"
	ActionCreateDirectory = "createDirectory"
)

type UploadInput struct {
	// TargetDirectory is the directory path where the file will be stored
	// This is a required field for all uploads
	TargetDirectory string `json:"targetDirectory"`
	// RelativePaths is now a slice of strings, corresponding to the order of files
	RelativePaths []string `json:"relativePaths"`
	// Note: All uploads are now handled directly from HTTP multipart/form-data requests
	// For both single and multiple file uploads: The file content must be provided in the 'file' field of the multipart form
}
type DownloadInput struct {
	TargetFilePath string `json:"targetFilePath"`
}
type ListInput DownloadInput
type RemoveInput DownloadInput

type CreateDirectoryInput struct {
	TargetDirectory string `json:"targetDirectory"`
}

type BaseHandler struct {
	Ctx                    context.Context
	RegistryManager        *registry.Manager
	GetRegistryAndRootPath func(namespace, name string) (string, string, error)
}

func (h BaseHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if err := h.do(rw, req); err != nil {
		logrus.Errorf("do action failed: %v", err)
		status := http.StatusInternalServerError
		var e *apierror.APIError
		if errors.As(err, &e) {
			status = e.Code.Status
		}
		utils.ResponseAPIError(rw, status, e)
		return
	}
	// if rw has content, it will be written to response writer and return 200, otherwise, return 204
	utils.ResponseOKWithNoContent(rw)
}

func (h BaseHandler) do(rw http.ResponseWriter, req *http.Request) error {
	vars := utils.EncodeVars(mux.Vars(req))

	if req.Method == http.MethodPost {
		return h.doPost(rw, req, vars)
	}

	return apierror.NewAPIError(validation.InvalidAction, fmt.Sprintf("Unsupported method %s", req.Method))
}

func (h BaseHandler) doPost(rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
	action := vars["action"]
	namespace, name := vars["namespace"], vars["name"]

	switch action {
	case ActionUpload:
		return h.upload(req, namespace, name)
	case ActionDownload:
		return h.download(rw, req, namespace, name)
	case ActionList:
		return h.list(rw, req, namespace, name)
	case ActionRemove:
		return h.remove(req, namespace, name)
	case ActionCreateDirectory:
		return h.createDirectory(req, namespace, name)
	default:
		return apierror.NewAPIError(validation.InvalidAction, fmt.Sprintf("Unsupported action %s", action))
	}
}

func (h BaseHandler) upload(req *http.Request, namespace, name string) error {
	// All uploads are now direct uploads from HTTP requests
	// Verify that this is a multipart form request
	contentType := req.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		return apierror.NewAPIError(validation.InvalidBodyContent, "Upload requires a multipart/form-data request")
	}

	// Parse the multipart form
	if err := req.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory
		return apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("Failed to parse multipart form: %v", err))
	}

	// Get the JSON data from the form
	jsonData := req.FormValue("data")
	if jsonData == "" {
		return apierror.NewAPIError(validation.InvalidBodyContent, "Missing 'data' field in multipart form")
	}

	// Decode the JSON data
	input := &UploadInput{}
	if err := json.Unmarshal([]byte(jsonData), input); err != nil {
		return apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("Failed to parse JSON data: %v", err))
	}

	b, rootPath, err := h.getBackendAndRootPath(namespace, name)
	if err != nil {
		return err
	}

	// Get all files from the multipart form
	form := req.MultipartForm
	if form == nil || form.File == nil {
		return apierror.NewAPIError(validation.InvalidBodyContent, "No files found in multipart form")
	}

	// Look for files with the field name 'file'
	files, ok := form.File["file"]
	if !ok || len(files) == 0 {
		return apierror.NewAPIError(validation.InvalidBodyContent, "No files found with field name 'file'")
	}

	// Process each file
	for i, fileHeader := range files {
		// Open the file
		file, err := fileHeader.Open()
		if err != nil {
			return apierror.NewAPIError(validation.InvalidBodyContent,
				fmt.Sprintf("Failed to open file %s: %v", fileHeader.Filename, err))
		}

		// Determine the filename
		fileName := fileHeader.Filename

		// Get the relative path for the file
		var relativePath string
		if i < len(input.RelativePaths) {
			relativePath = input.RelativePaths[i]
		}

		// Construct the destination path
		destPath := path.Join(rootPath, input.TargetDirectory, relativePath, fileName)

		// Upload the file
		err = b.UploadFromReader(h.Ctx, file, destPath, fileHeader.Size, fileHeader.Header.Get("Content-Type"))
		file.Close() // nolint:errcheck

		if err != nil {
			return fmt.Errorf("upload file %s failed: %w", fileName, err)
		}
	}

	return nil
}

func (h BaseHandler) download(rw http.ResponseWriter, req *http.Request, namespace, name string) error {
	input := &DownloadInput{}

	err := decodeAndValidateInput(req, input, input.TargetFilePath)
	if err != nil {
		return err
	}

	b, rootPath, err := h.getBackendAndRootPath(namespace, name)
	if err != nil {
		return err
	}

	objectName := path.Join(rootPath, input.TargetFilePath)

	fileInfo, err := b.List(h.Ctx, objectName, false, false)
	if err != nil {
		return fmt.Errorf("failed to list %s: %w", objectName, err)
	}
	if len(fileInfo) == 0 {
		return apierror.NewAPIError(validation.NotFound, fmt.Sprintf("target %s not found", objectName))
	}
	// if fileInfo[0].Size == 0, it means the objectName is a directory
	if fileInfo[0].Size == 0 {
		zipFileName := fmt.Sprintf("%s.zip", path.Base(input.TargetFilePath))
		rw.Header().Set("Content-Type", "application/zip")
		rw.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
			url.QueryEscape(zipFileName), url.QueryEscape(zipFileName)))
	} else {
		rw.Header().Set("Content-Type", fileInfo[0].ContentType)
		rw.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
			url.QueryEscape(fileInfo[0].Name), url.QueryEscape(fileInfo[0].Name)))
	}

	if err := b.Download(h.Ctx, objectName, rw); err != nil {
		return fmt.Errorf("download file %s failed: %w", input.TargetFilePath, err)
	}

	return nil
}

func (h BaseHandler) list(rw http.ResponseWriter, req *http.Request, namespace, name string) error {
	input := &ListInput{}
	err := decodeAndValidateInput(req, input, input.TargetFilePath)
	if err != nil {
		return err
	}

	b, rootPath, err := h.getBackendAndRootPath(namespace, name)
	if err != nil {
		return err
	}

	objectName := path.Join(rootPath, input.TargetFilePath)
	output, err := b.List(h.Ctx, objectName, false, true)
	if err != nil {
		return fmt.Errorf("failed to list %s: %w", input.TargetFilePath, err)
	}

	utils.ResponseOKWithBody(rw, output)

	return nil
}

func (h BaseHandler) remove(req *http.Request, namespace, name string) error {
	input := &RemoveInput{}

	err := decodeAndValidateInput(req, input, input.TargetFilePath)
	if err != nil {
		return err
	}

	b, rootPath, err := h.getBackendAndRootPath(namespace, name)
	if err != nil {
		return err
	}

	objectName := path.Join(rootPath, input.TargetFilePath)
	if err := b.Delete(h.Ctx, objectName); err != nil {
		return fmt.Errorf("remove file %s failed: %v", objectName, err)
	}

	return nil
}

func (h BaseHandler) createDirectory(req *http.Request, namespace, name string) error {
	input := &CreateDirectoryInput{}

	err := decodeAndValidateInput(req, input, input.TargetDirectory)
	if err != nil {
		return err
	}

	ctx := req.Context()
	b, rootPath, err := h.getBackendAndRootPath(namespace, name)
	if err != nil {
		return err
	}

	directory := path.Join(rootPath, input.TargetDirectory)
	if err := b.CreateDirectory(ctx, directory); err != nil {
		return fmt.Errorf("create directory %s failed: %w", directory, err)
	}

	return nil
}

func decodeAndValidateInput(req *http.Request, input interface{}, pathField string) error {
	if err := json.NewDecoder(req.Body).Decode(input); err != nil {
		return apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("Failed to parse body: %v", err))
	}

	if !isValidPath(pathField) {
		return apierror.NewAPIError(validation.InvalidBodyContent, "Invalid path")
	}

	return nil
}

func (h BaseHandler) getBackendAndRootPath(namespace, name string) (backend.Backend, string, error) {
	reg, rootPath, err := h.GetRegistryAndRootPath(namespace, name)
	if err != nil {
		return nil, "", fmt.Errorf("get registry and root path failed: %w", err)
	}

	b, err := h.RegistryManager.NewBackendFromRegistry(h.Ctx, reg)
	if err != nil {
		return nil, "", fmt.Errorf("new backend for registry %s failed: %w", reg, err)
	}

	return b, rootPath, nil
}

// isValidPath checks if the path is valid and doesn't contain directory traversal attempts
func isValidPath(p string) bool {
	// Prevent paths with ".." which could lead to directory traversal
	if strings.Contains(p, "..") {
		return false
	}

	// Prevent absolute paths
	if strings.HasPrefix(p, "/") {
		return false
	}

	return true
}
