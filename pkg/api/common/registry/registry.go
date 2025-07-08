package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/sirupsen/logrus"

	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/registry/backend"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

const (
	ActionUpload               = "upload"
	ActionDownload             = "download"
	ActionList                 = "list"
	ActionRemove               = "remove"
	ActionCreateDirectory      = "createDirectory"
	ActionGeneratePresignedURL = "generatePresignedURL"
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

type PostHook func(req *http.Request, b backend.Backend) error

type CreateDirectoryInput struct {
	TargetDirectory string `json:"targetDirectory"`
}

type GeneratePresignedURLInput struct {
	ObjectName  string `json:"objectName"`            // The object name/path in storage
	Operation   string `json:"operation"`             // "upload" or "download"
	ContentType string `json:"contentType,omitempty"` // Content type for upload (optional)
	ExpiryHours int    `json:"expiryHours,omitempty"` // Expiry time in hours (default: 1 hour)
}

type GeneratePresignedURLOutput struct {
	PresignedURL string `json:"presignedURL"`
	ExpiresAt    string `json:"expiresAt"`
	Operation    string `json:"operation"`
}

type Progress struct {
	DestPath  string `json:"destPath"`
	TotalSize int64  `json:"totalSize"`
	ReadSize  int64  `json:"readSize"`
}

// ResponseWriterSync wraps an http.ResponseWriter with a mutex for thread safety
type ResponseWriterSync struct {
	sync.Mutex
	rw http.ResponseWriter
}

type BaseHandler struct {
	Ctx                    context.Context
	RegistryManager        *registry.Manager
	GetRegistryAndRootPath func(namespace, name string) (string, string, error)

	PostHooks map[string]PostHook
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
		return h.upload(rw, req, namespace, name)
	case ActionDownload:
		return h.download(rw, req, namespace, name)
	case ActionList:
		return h.list(rw, req, namespace, name)
	case ActionRemove:
		return h.remove(req, namespace, name)
	case ActionCreateDirectory:
		return h.createDirectory(req, namespace, name)
	case ActionGeneratePresignedURL:
		return h.generatePresignedURL(rw, req, namespace, name)
	default:
		return apierror.NewAPIError(validation.InvalidAction, fmt.Sprintf("Unsupported action %s", action))
	}
}

func (h BaseHandler) upload(rw http.ResponseWriter, req *http.Request, namespace, name string) error {
	// All uploads are now direct uploads from HTTP requests
	// Verify that this is a multipart form request
	contentType := req.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		return apierror.NewAPIError(validation.InvalidBodyContent, "Upload requires a multipart/form-data request")
	}

	// Parse the multipart form with increased memory limit for large files
	// Use 128MB max memory to handle larger files better
	if err := req.ParseMultipartForm(128 << 20); err != nil { // 128MB max memory
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

	// Use Server-Sent Events (SSE) to send progress updates
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")

	syncWriter := &ResponseWriterSync{rw: rw}

	errors := make(chan error, len(files))

	// Add a WaitGroup to wait for all uploads to complete
	var wg sync.WaitGroup

	// Process each file
	for i := range files {
		// Get the relative path for the file
		var relativePath string
		if i < len(input.RelativePaths) {
			relativePath = input.RelativePaths[i]
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errors <- h.uploadOneFile(syncWriter, b, files[i], path.Join(rootPath, input.TargetDirectory, relativePath))
		}(i)
	}

	// Wait for all uploads to complete in a separate goroutine
	go func() {
		wg.Wait()
		close(errors)
	}()

	// Collect any errors that occurred during upload
	var uploadErrors []error
	for err := range errors {
		if err != nil {
			uploadErrors = append(uploadErrors, err)
		}
	}

	// If any errors occurred, return the first one
	if len(uploadErrors) > 0 {
		return uploadErrors[0]
	}

	if hook, ok := h.PostHooks[ActionUpload]; ok {
		if err := hook(req, b); err != nil {
			return fmt.Errorf("execute post hook failed: %w", err)
		}
	}
	logrus.Infof("upload %s successfully", name)

	return nil
}

func (h BaseHandler) uploadOneFile(rw *ResponseWriterSync, b backend.Backend, fileHeader *multipart.FileHeader, targetPath string) error {
	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		return apierror.NewAPIError(validation.InvalidBodyContent,
			fmt.Sprintf("Failed to open file %s: %v", fileHeader.Filename, err))
	}
	defer func() {
		if err := file.Close(); err != nil {
			logrus.Errorf("Failed to close file %s: %v", fileHeader.Filename, err)
		}
	}() // Ensure file is always closed

	// Construct the destination path
	destPath := path.Join(targetPath, fileHeader.Filename)

	// Use larger buffer for progress channel to prevent blocking
	processChan := make(chan int64, 100)
	reader := backend.NewProgressReader(file, fileHeader.Size, processChan)

	// Context for canceling the progress reporting
	ctx, cancel := context.WithCancel(h.Ctx)
	defer cancel() // Ensure context is always canceled

	// Start progress reporting in background
	go reportProgress(ctx, rw, processChan, fileHeader.Size, destPath)

	// Upload the file with streaming support
	err = b.UploadFromReader(h.Ctx, reader, destPath, fileHeader.Size, fileHeader.Header.Get("Content-Type"))
	if err != nil {
		return fmt.Errorf("upload file %s failed: %w", destPath, err)
	}

	return nil
}

func reportProgress(ctx context.Context, rw *ResponseWriterSync, processChan chan int64, totalSize int64, destPath string) {
	// Use a buffered channel to prevent blocking
	progressBuffer := make(chan int64, 100)

	// Start a goroutine to drain the original channel
	go func() {
		defer close(progressBuffer)
		for {
			select {
			case progress, ok := <-processChan:
				if !ok {
					return
				}
				// Non-blocking send to buffer
				select {
				case progressBuffer <- progress:
				default:
					// Drop progress update if buffer is full
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Report progress from buffer
	for {
		select {
		case progress, ok := <-progressBuffer:
			if !ok {
				return
			}
			p := &Progress{
				DestPath:  destPath,
				TotalSize: totalSize,
				ReadSize:  progress,
			}
			data, err := json.Marshal(p)
			if err != nil {
				logrus.Errorf("marshal progress %+v failed: %v", p, err)
				continue
			}
			if _, err := fmt.Fprintf(rw, "data: %s\n", data); err != nil {
				logrus.Errorf("stream progress write failed: %v", err)
				return
			}
			rw.Flush()

		case <-ctx.Done():
			return
		}
	}
}

func (h BaseHandler) download(rw http.ResponseWriter, req *http.Request, namespace, name string) error {
	input := &DownloadInput{}

	err := decodeAndValidateInput(req, input, input.TargetFilePath)
	if err != nil {
		return apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("failed to parse body: %v", err))
	}

	b, rootPath, err := h.getBackendAndRootPath(namespace, name)
	if err != nil {
		return apierror.NewAPIError(validation.ServerError, fmt.Sprintf("get backend failed: %v", err))
	}

	objectName := path.Join(rootPath, input.TargetFilePath)

	fileInfo, err := b.List(h.Ctx, objectName, false, false)
	if err != nil {
		return apierror.NewAPIError(validation.ServerError, fmt.Sprintf("list %s failed: %v", objectName, err))
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
		return apierror.NewAPIError(validation.ServerError, fmt.Sprintf("download %s failed: %v", objectName, err))
	}

	if hook, ok := h.PostHooks[ActionDownload]; ok {
		return hook(req, b)
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

	if hook, ok := h.PostHooks[ActionList]; ok {
		return hook(req, b)
	}

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

	if hook, ok := h.PostHooks[ActionRemove]; ok {
		return hook(req, b)
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

	if hook, ok := h.PostHooks[ActionCreateDirectory]; ok {
		return hook(req, b)
	}

	return nil
}

func (h BaseHandler) generatePresignedURL(rw http.ResponseWriter, req *http.Request, namespace, name string) error {
	input := &GeneratePresignedURLInput{}

	if err := json.NewDecoder(req.Body).Decode(input); err != nil {
		return apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("Failed to parse body: %v", err))
	}

	// Validate input
	if input.ObjectName == "" {
		return apierror.NewAPIError(validation.InvalidBodyContent, "ObjectName is required")
	}
	if input.Operation != "upload" && input.Operation != "download" {
		return apierror.NewAPIError(validation.InvalidBodyContent, "Operation must be 'upload' or 'download'")
	}
	if !isValidPath(input.ObjectName) {
		return apierror.NewAPIError(validation.InvalidBodyContent, "Invalid object name")
	}

	b, rootPath, err := h.getBackendAndRootPath(namespace, name)
	if err != nil {
		return err
	}

	// Set default expiry if not provided
	expiryHours := input.ExpiryHours
	if expiryHours <= 0 {
		expiryHours = 1 // Default 1 hour
	}
	expiry := time.Duration(expiryHours) * time.Hour

	objectName := path.Join(rootPath, input.ObjectName)
	var presignedURL string

	switch input.Operation {
	case "upload":
		presignedURL, err = b.GeneratePresignedUploadURL(h.Ctx, objectName, expiry, input.ContentType)
	case "download":
		presignedURL, err = b.GeneratePresignedDownloadURL(h.Ctx, objectName, expiry)
	}

	if err != nil {
		return apierror.NewAPIError(validation.ServerError, fmt.Sprintf("Failed to generate presigned URL: %v", err))
	}

	output := &GeneratePresignedURLOutput{
		PresignedURL: presignedURL,
		ExpiresAt:    time.Now().Add(expiry).Format(time.RFC3339),
		Operation:    input.Operation,
	}

	utils.ResponseOKWithBody(rw, output)
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

// Write safely writes to the underlying ResponseWriter with mutex protection
func (r *ResponseWriterSync) Write(p []byte) (n int, err error) {
	r.Lock()
	defer r.Unlock()
	return r.rw.Write(p)
}

// Flush safely flushes the underlying ResponseWriter with mutex protection
func (r *ResponseWriterSync) Flush() {
	r.Lock()
	defer r.Unlock()
	if flusher, ok := r.rw.(http.Flusher); ok {
		flusher.Flush()
	}
}
