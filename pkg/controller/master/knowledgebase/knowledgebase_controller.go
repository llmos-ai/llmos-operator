package knowledgebase

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/textsplitter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	agentv1 "github.com/llmos-ai/llmos-operator/pkg/apis/agent.llmos.ai/v1"
	ctlagentv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/agent.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	vd "github.com/llmos-ai/llmos-operator/pkg/vectordatabase"
	"github.com/llmos-ai/llmos-operator/pkg/vectordatabase/helper"
)

type handler struct {
	ctx context.Context

	knowledgeBaseClient ctlagentv1.KnowledgeBaseClient
	dataCollectionCache ctlagentv1.DataCollectionCache

	rm *registry.Manager
}

func Register(_ context.Context, mgmt *config.Management, _ config.Options) error {
	registries := mgmt.LLMFactory.Ml().V1().Registry()
	secrets := mgmt.CoreFactory.Core().V1().Secret()
	datacollections := mgmt.AgentFactory.Agent().V1().DataCollection()
	knowledgebases := mgmt.AgentFactory.Agent().V1().KnowledgeBase()

	h := handler{
		ctx:                 mgmt.Ctx,
		knowledgeBaseClient: knowledgebases,
		dataCollectionCache: datacollections.Cache(),
	}
	h.rm = registry.NewManager(secrets.Cache().Get, registries.Cache().Get)

	knowledgebases.OnChange(mgmt.Ctx, "knownledgebase.SyncFiles", h.SyncFiles)
	knowledgebases.OnChange(mgmt.Ctx, "knownledgebase.SyncObjects", h.SyncObjects)
	knowledgebases.OnRemove(mgmt.Ctx, "knownledgebase.OnRemove", h.OnRemove)

	return nil
}

func (h *handler) SyncFiles(_ string, kb *agentv1.KnowledgeBase) (*agentv1.KnowledgeBase, error) {
	if kb == nil || kb.DeletionTimestamp != nil {
		return kb, nil
	}

	filesToRemove, filesToAdd := deltaFiles(kb)

	if len(filesToRemove) == 0 && len(filesToAdd) == 0 {
		return kb, nil
	}

	kbCopy := kb.DeepCopy()

	h.deleteFiles(kbCopy, filesToRemove)
	err := h.addFiles(kbCopy, filesToAdd)
	return h.updateKnowledgeBaseStatus(kbCopy, kb, err)
}

func (h *handler) SyncObjects(_ string, kb *agentv1.KnowledgeBase) (*agentv1.KnowledgeBase, error) {
	if kb == nil || kb.DeletionTimestamp != nil {
		return kb, nil
	}

	c, err := helper.NewVectorDatabaseClient(kb.Spec.EmbeddingModel)
	if err != nil {
		return kb, fmt.Errorf("new vector database client with embedding model %s: %w", kb.Spec.EmbeddingModel, err)
	}

	kbCopy := kb.DeepCopy()
	if err = h.ensureCollectionExists(c, kbCopy); err != nil {
		return h.updateKnowledgeBaseStatus(kbCopy, kb, err)
	}

	err = h.syncObjects(kbCopy, c)
	return h.updateKnowledgeBaseStatus(kbCopy, kb, err)
}

func (h *handler) OnRemove(_ string, kb *agentv1.KnowledgeBase) (*agentv1.KnowledgeBase, error) {
	if kb == nil || kb.DeletionTimestamp == nil {
		return nil, nil
	}

	c, err := helper.NewVectorDatabaseClient(kb.Spec.EmbeddingModel)
	if err != nil {
		return nil, fmt.Errorf("new vector database client with embedding model %s: %w", kb.Spec.EmbeddingModel, err)
	}

	// Convert name to valid Weaviate class name
	if err := c.DeleteCollection(h.ctx, kb.Status.ClassName); err != nil {
		return nil, fmt.Errorf("failed to delete collection %s: %w", kb.Status.ClassName, err)
	}

	return kb, nil
}

func (h *handler) ensureCollectionExists(c vd.Client, kb *agentv1.KnowledgeBase) error {
	// Convert name to valid Weaviate class name
	// Weaviate class names must start with uppercase letter and contain only letters, numbers, and underscores
	className := toValidWeaviateClassName(kb.Name)

	exists, err := c.CollectionExists(h.ctx, className)
	if err != nil {
		return fmt.Errorf("failed to check collection %s exists: %w", className, err)
	}
	if !exists {
		if err := c.CreateCollection(h.ctx, className); err != nil {
			return fmt.Errorf("failed to create collection %s: %w", className, err)
		}
	}

	kb.Status.ClassName = className
	agentv1.Ready.True(kb)

	return nil
}

// deltaFiles compares kb.Spec.ImportingFils and kb.Status.ImportedFiles to get the added files and deleted files
// return files to remove and files to add
// filesToRemove: []uid
// filesToAdd: [](dataCollectionName, uid)
func deltaFiles(kb *agentv1.KnowledgeBase) ([]string, []agentv1.ImportingFile) {
	filesToRemove := make([]string, 0, len(kb.Status.ImportedFiles))
	filesToAdd := make([]agentv1.ImportingFile, 0, len(kb.Spec.ImportingFiles))

	importingFiles := make(map[string]string, len(kb.Spec.ImportingFiles))
	for _, file := range kb.Spec.ImportingFiles {
		importingFiles[file.UID] = file.DataCollectionName
	}

	for _, file := range kb.Status.ImportedFiles {
		if _, ok := importingFiles[file.UID]; ok {
			delete(importingFiles, file.UID)
		} else {
			filesToRemove = append(filesToRemove, file.UID)
		}
	}

	for uid, dataCollectionName := range importingFiles {
		filesToAdd = append(filesToAdd, agentv1.ImportingFile{
			DataCollectionName: dataCollectionName,
			UID:                uid,
		})
	}

	return filesToRemove, filesToAdd
}

func (h *handler) deleteFiles(kb *agentv1.KnowledgeBase, filesToRemove []string) {
	if len(filesToRemove) == 0 {
		return
	}

	fileMap := make(map[string]struct{}, len(filesToRemove))
	for _, uid := range filesToRemove {
		fileMap[uid] = struct{}{}
	}
	for i, file := range kb.Status.ImportedFiles {
		if _, ok := fileMap[file.UID]; ok {
			agentv1.DeleteObject.True(&kb.Status.ImportedFiles[i])
		}
	}
}

func (h *handler) addFiles(kb *agentv1.KnowledgeBase, filesToAdd []agentv1.ImportingFile) error {
	if len(filesToAdd) == 0 {
		return nil
	}

	for _, file := range filesToAdd {
		importedFile, err := h.toImportedFile(kb, file)
		if err != nil {
			return fmt.Errorf("failed to get file info of UID %s: %w", file.UID, err)
		}
		kb.Status.ImportedFiles = append(kb.Status.ImportedFiles, *importedFile)
	}

	return nil
}

func (h *handler) syncObjects(kb *agentv1.KnowledgeBase, c vd.Client) error {
	importedFiles := make([]agentv1.ImportedFile, 0, len(kb.Status.ImportedFiles))

	for _, file := range kb.Status.ImportedFiles {
		if agentv1.Ready.IsTrue(file) {
			importedFiles = append(importedFiles, file)
			continue
		}

		if agentv1.DeleteObject.IsTrue(file) {
			if _, err := c.DeleteObjects(h.ctx, kb.Status.ClassName, file.UID); err != nil {
				return fmt.Errorf("delete objects with uid %s: %w", file.UID, err)
			}
		}

		if agentv1.InsertObject.IsTrue(file) {
			objects, err := h.getObjectsPerFile(kb, file)
			if err != nil {
				return fmt.Errorf("failed to get objects of file %s: %w", file.UID, err)
			}
			logrus.Infof("inserting objects of file %s, object number: %d", file.UID, len(objects))
			if err := c.InsertObjects(h.ctx, kb.Status.ClassName, objects); err != nil {
				return fmt.Errorf("failed to insert objects of file %s: %w", file.UID, err)
			}
			logrus.Infof("insert objects of file %s successfully", file.UID)
			agentv1.Ready.True(&file)
			file.ImportedTime = metav1.NewTime(time.Now())
			importedFiles = append(importedFiles, file)
		}
	}

	kb.Status.ImportedFiles = importedFiles

	return nil
}

func (h *handler) toImportedFile(kb *agentv1.KnowledgeBase, file agentv1.ImportingFile) (*agentv1.ImportedFile, error) {
	dc, err := h.dataCollectionCache.Get(kb.Namespace, file.DataCollectionName)
	if err != nil {
		return nil, fmt.Errorf("get data collection %s: %w", file.DataCollectionName, err)
	}
	// Get file info
	for _, f := range dc.Status.Files {
		if f.UID != file.UID {
			continue
		}

		importedFile := &agentv1.ImportedFile{
			UID:                file.UID,
			DataCollectionName: file.DataCollectionName,
			FileInfo:           f,
			ImportedTime:       metav1.NewTime(time.Now()),
		}
		agentv1.InsertObject.True(importedFile)
		return importedFile, nil
	}

	return nil, fmt.Errorf("file %s not found in data collection %s", file.UID, file.DataCollectionName)
}

func (h *handler) getObjectsPerFile(kb *agentv1.KnowledgeBase, file agentv1.ImportedFile) ([]vd.Document, error) {
	objects := make([]vd.Document, 0)

	dc, err := h.dataCollectionCache.Get(kb.Namespace, file.DataCollectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get data collection %s/%s: %w", kb.Namespace, file.DataCollectionName, err)
	}

	chunks, err := h.getChunks(dc.Spec.Registry, file.FileInfo.Path, kb.Spec.ChunkingConfig)
	if err != nil {
		return nil, fmt.Errorf("get chunks for file %s in registry %s: %w", file.FileInfo.Path, dc.Spec.Registry, err)
	}

	for i, chunk := range chunks {
		objects = append(objects, vd.Document{
			BaseDocument: vd.BaseDocument{
				UID:      file.UID,
				Document: file.FileInfo.Name,
				Index:    i,
				// TODO: Keywords
				Keywords:  dc.Name,
				Content:   chunk,
				Timestamp: metav1.NewTime(time.Now()).String(),
			},
		})
	}

	return objects, nil
}

func (h *handler) updateKnowledgeBaseStatus(kbCopy, kb *agentv1.KnowledgeBase,
	err error) (*agentv1.KnowledgeBase, error) {
	if err == nil {
		agentv1.Ready.True(kbCopy)
		agentv1.Ready.Message(kbCopy, "")
	} else {
		agentv1.Ready.False(kbCopy)
		agentv1.Ready.Message(kbCopy, err.Error())
	}

	// don't update when no change happens
	if reflect.DeepEqual(kbCopy.Status, kb.Status) {
		return kbCopy, err
	}

	updatedKB, updateErr := h.knowledgeBaseClient.UpdateStatus(kbCopy)
	if updateErr != nil {
		return nil, fmt.Errorf("update knowledge base status failed: %w", updateErr)
	}
	return updatedKB, err
}

// toValidWeaviateClassName converts a string to a valid Weaviate class name
// Weaviate class names must:
// - Start with an uppercase letter
// - Contain only letters, numbers, and underscores
// - Follow the regex pattern: /^[A-Z][_0-9A-Za-z]*$/
func toValidWeaviateClassName(name string) string {
	if name == "" {
		return "DefaultClass"
	}

	// Replace hyphens and other invalid characters with underscores
	invalidChars := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	cleanName := invalidChars.ReplaceAllString(name, "_")

	// Remove leading and trailing underscores
	cleanName = strings.Trim(cleanName, "_")

	// If empty after cleaning, use default
	if cleanName == "" {
		return "DefaultClass"
	}

	// Ensure first character is uppercase letter
	if len(cleanName) > 0 {
		firstChar := rune(cleanName[0])
		if unicode.IsDigit(firstChar) || firstChar == '_' {
			// Prepend with a letter if starts with digit or underscore
			cleanName = "Class_" + cleanName
		} else {
			// Convert first character to uppercase
			cleanName = strings.ToUpper(string(firstChar)) + cleanName[1:]
		}
	}

	return cleanName
}

func (h *handler) getChunks(registry, path string, chunkingConfig agentv1.ChunkingConfig) ([]string, error) {
	splitter := textsplitter.NewMarkdownTextSplitter(
		textsplitter.WithChunkSize(chunkingConfig.Size),
		textsplitter.WithChunkOverlap(chunkingConfig.Overlap),
	)

	// Download file from registry
	backend, err := h.rm.NewBackendFromRegistry(h.ctx, registry)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend from registry %s: %w", registry, err)
	}

	// Create a buffer to store the downloaded file content
	var buf bytes.Buffer
	if err := backend.Download(h.ctx, path, &buf); err != nil {
		return nil, fmt.Errorf("failed to download file %s from registry %s: %w", path, registry, err)
	}

	content := buf.String()

	return splitter.SplitText(content)
}
