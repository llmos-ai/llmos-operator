package registry

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gorilla/mux"
	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/types"
	corev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/sirupsen/logrus"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/registry/backend"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

const (
	modelVersionAPIType   = "ml.llmos.ai.modelversions"
	datasetVersionAPIType = "ml.llmos.ai.datasetversions"
)

func Formatter(request *types.APIRequest, resource *types.RawResource) {
	resource.Actions = make(map[string]string, 1)
	if request.AccessControl.CanUpdate(request, resource.APIObject, resource.Schema) != nil {
		return
	}
	resource.AddAction(request, ActionUpload)
	resource.AddAction(request, ActionDownload)
	resource.AddAction(request, ActionList)
	resource.AddAction(request, ActionRemove)
	resource.AddAction(request, ActionCreateDirectory)
}

type Handler struct {
	modelCache          ctlmlv1.ModelCache
	datasetCache        ctlmlv1.DatasetCache
	registryCache       ctlmlv1.RegistryCache
	secretCache         corev1.SecretCache
	modelVersionCache   ctlmlv1.ModelVersionCache
	datasetVersionCache ctlmlv1.DatasetVersionCache

	rm *registry.Manager
}

func NewHandler(scaled *config.Scaled) Handler {
	h := Handler{
		modelCache:          scaled.Management.LLMFactory.Ml().V1().Model().Cache(),
		datasetCache:        scaled.Management.LLMFactory.Ml().V1().Dataset().Cache(),
		registryCache:       scaled.Management.LLMFactory.Ml().V1().Registry().Cache(),
		secretCache:         scaled.CoreFactory.Core().V1().Secret().Cache(),
		modelVersionCache:   scaled.Management.LLMFactory.Ml().V1().ModelVersion().Cache(),
		datasetVersionCache: scaled.Management.LLMFactory.Ml().V1().DatasetVersion().Cache(),
	}
	h.rm = registry.NewManager(h.secretCache, h.registryCache, h.modelCache, h.datasetCache)

	return h
}

func (h Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if err := h.do(rw, req); err != nil {
		logrus.Error(err)
		status := http.StatusInternalServerError
		var e *apierror.APIError
		if errors.As(err, &e) {
			status = e.Code.Status
		}
		utils.ResponseAPIError(rw, status, e)
		return
	}
	utils.ResponseOKWithNoContent(rw)
}

func (h Handler) do(rw http.ResponseWriter, req *http.Request) error {
	vars := utils.EncodeVars(mux.Vars(req))

	if req.Method == http.MethodPost {
		return h.doPost(rw, req, vars)
	}

	return apierror.NewAPIError(validation.InvalidAction, fmt.Sprintf("Unsupported method %s", req.Method))
}

func (h Handler) doPost(rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
	action := vars["action"]
	namespace, name := vars["namespace"], vars["name"]
	typeName := vars["type"]

	switch action {
	case ActionUpload:
		return h.upload(req, typeName, namespace, name)
	case ActionDownload:
		return h.download(rw, req, typeName, namespace, name)
	case ActionList:
		return h.list(rw, req, typeName, namespace, name)
	case ActionRemove:
		return h.remove(req, typeName, namespace, name)
	case ActionCreateDirectory:
		return h.createDirectory(req, typeName, namespace, name)
	default:
		return apierror.NewAPIError(validation.InvalidAction, fmt.Sprintf("Unsupported action %s", action))
	}
}

func (h Handler) upload(req *http.Request, typeName, namespace, name string) error {
	input := &UploadInput{}
	if err := json.NewDecoder(req.Body).Decode(input); err != nil {
		return apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("Failed to parse body: %v", err))
	}

	b, rootPath, err := h.getBackendAndRootPath(typeName, namespace, name)
	if err != nil {
		return fmt.Errorf("get backend and root path failed: %v", err)
	}

	fileInfo, err := os.Stat(input.SourceFilePath)
	if err != nil {
		return fmt.Errorf("stat file %s failed: %v", input.SourceFilePath, err)
	}

	if fileInfo.IsDir() {
		return h.uploadDirectory(b, input.SourceFilePath, path.Join(rootPath, input.TargetDirectory))
	}
	return h.uploadFile(b, input.SourceFilePath, path.Join(rootPath, input.TargetDirectory, fileInfo.Name()))
}

func (h Handler) uploadFile(backend backend.Backend, sourcePath, objectName string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open file %s failed: %v", sourcePath, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat file %s failed: %v", sourcePath, err)
	}

	contentType, err := mimetype.DetectReader(file)
	if err != nil {
		return fmt.Errorf("detect file %s mimetype failed: %v", sourcePath, err)
	}
	// reset file position because mimetype.DetectReader read the file
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("reset file position failed: %v", err)
	}

	if err := backend.Upload(objectName, file, fileInfo.Size(), contentType.String()); err != nil {
		return fmt.Errorf("upload file %s failed: %v", sourcePath, err)
	}

	return nil
}

func (h Handler) uploadDirectory(backend backend.Backend, sourceDir, targetPrefix string) error {
	dirName := path.Base(sourceDir)
	return filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, filePath)
		if err != nil {
			return fmt.Errorf("get relative path failed: %v", err)
		}

		objectName := path.Join(targetPrefix, dirName, relPath)

		return h.uploadFile(backend, filePath, objectName)
	})
}

func (h Handler) download(rw http.ResponseWriter, req *http.Request, typeName, namespace, name string) error {
	input := &DownloadInput{}
	if err := json.NewDecoder(req.Body).Decode(input); err != nil {
		return apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("Failed to parse body: %v", err))
	}

	b, rootPath, err := h.getBackendAndRootPath(typeName, namespace, name)
	if err != nil {
		return fmt.Errorf("get backend and root path failed: %v", err)
	}

	objectName := path.Join(rootPath, input.TargetFilePath)
	files, err := b.List(objectName, true, true)
	if err != nil {
		return fmt.Errorf("list file %s failed: %v", objectName, err)
	}
	if len(files) == 1 && files[0].Path == objectName {
		return h.downloadFile(rw, b, files[0])
	}

	return h.downloadDirectory(rw, b, objectName, files)
}

func (h Handler) downloadFile(rw http.ResponseWriter, backend backend.Backend, file backend.FileInfo) error {
	fileName := path.Base(file.Path)
	rw.Header().Set("Content-Type", file.ContentType)
	rw.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
		url.QueryEscape(fileName), url.QueryEscape(fileName)))

	buf := bufio.NewWriter(rw)
	defer buf.Flush()

	return backend.Download(file.Path, buf)
}

func (h Handler) downloadDirectory(rw http.ResponseWriter, backend backend.Backend, directory string, files []backend.FileInfo) error {
	zipFileName := path.Base(directory) + ".zip"
	rw.Header().Set("Content-Type", "application/zip")
	rw.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
		url.QueryEscape(zipFileName), url.QueryEscape(zipFileName)))

	zw := zip.NewWriter(rw)
	defer zw.Close()

	for _, file := range files {
		relPath := strings.TrimPrefix(file.Path, directory)
		fw, err := zw.Create(relPath)
		if err != nil {
			return fmt.Errorf("create zip entry failed: %v", err)
		}

		if err := backend.Download(file.Path, fw); err != nil {
			return fmt.Errorf("download file %s failed: %v", file.Name, err)
		}
	}

	return nil
}

func (h Handler) list(rw http.ResponseWriter, req *http.Request, typeName, namespace, name string) error {
	input := &ListInput{}
	if err := json.NewDecoder(req.Body).Decode(input); err != nil {
		return apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("Failed to parse body: %v", err))
	}

	b, rootPath, err := h.getBackendAndRootPath(typeName, namespace, name)
	if err != nil {
		return fmt.Errorf("get backend and root path failed: %v", err)
	}

	objectName := path.Join(rootPath, input.TargetFilePath)
	output, err := b.List(objectName, false, true)
	if err != nil {
		return fmt.Errorf("list file %s failed: %v", input.TargetFilePath, err)
	}

	utils.ResponseOKWithBody(rw, output)

	return nil
}

func (h Handler) remove(req *http.Request, typeName, namespace, name string) error {
	input := &RemoveInput{}
	if err := json.NewDecoder(req.Body).Decode(input); err != nil {
		return apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("Failed to parse body: %v", err))
	}

	b, rootPath, err := h.getBackendAndRootPath(typeName, namespace, name)
	if err != nil {
		return fmt.Errorf("get backend and root path failed: %v", err)
	}

	objectName := path.Join(rootPath, input.TargetFilePath)
	if err := b.Delete(objectName); err != nil {
		return fmt.Errorf("remove file %s failed: %v", objectName, err)
	}

	return nil
}

func (h Handler) getBackendAndRootPath(typeName, namespace, name string) (backend.Backend, string, error) {
	var registry, rootPath string
	switch typeName {
	case modelVersionAPIType:
		v, err := h.modelVersionCache.Get(namespace, name)
		if err != nil {
			return nil, "", fmt.Errorf("get model version %s/%s of model %s failed: %w", namespace, name, v.Spec.Model, err)
		}
		if !mlv1.Ready.IsTrue(v) {
			return nil, "", fmt.Errorf("model version %s/%s is not ready", namespace, name)
		}
		registry, rootPath = v.Status.Registry, v.Status.RootPath

	case datasetVersionAPIType:
		v, err := h.datasetVersionCache.Get(namespace, name)
		if err != nil {
			return nil, "", fmt.Errorf("get dataset version %s/%s of dataset %s failed: %w", namespace, name, v.Spec.Dataset, err)
		}
		if !mlv1.Ready.IsTrue(v) {
			return nil, "", fmt.Errorf("dataset version %s/%s is not ready", namespace, name)
		}
		registry, rootPath = v.Status.Registry, v.Status.RootPath

	default:
		return nil, "", fmt.Errorf("unsupported type %s", typeName)
	}

	b, err := h.rm.NewBackendFromRegistry(registry)
	if err != nil {
		return nil, "", fmt.Errorf("new backend for registry %s failed: %w", registry, err)
	}
	return b, rootPath, nil
}

func (h Handler) createDirectory(req *http.Request, typeName, namespace, name string) error {
	input := &CreateDirectoryInput{}

	if err := json.NewDecoder(req.Body).Decode(input); err != nil {
		return apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("Failed to parse body: %v", err))
	}

	b, rootPath, err := h.getBackendAndRootPath(typeName, namespace, name)
	if err != nil {
		return fmt.Errorf("get backend and root path failed: %v", err)
	}

	directory := path.Join(rootPath, input.TargetDirectory)
	if err := b.CreateDirectory(directory); err != nil {
		return fmt.Errorf("create directory %s failed: %w", directory, err)
	}

	return nil
}
