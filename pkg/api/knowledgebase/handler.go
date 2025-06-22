package knowledgebase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"

	ctlagentv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/agent.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
	"github.com/llmos-ai/llmos-operator/pkg/vectordatabase/helper"
)

const (
	ActionSearch      = "search"
	ActionListObjects = "listObjects"
)

type SearchInput struct {
	Query     string  `json:"query"`
	Threshold float64 `json:"threshold"`
	Limit     int     `json:"limit"`
}

type ListObjectsInput struct {
	UID    string `json:"uid"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

type Handler struct {
	ctx                context.Context
	knowledgeBaseCache ctlagentv1.KnowledgeBaseCache
}

func NewHandler(scaled *config.Scaled) Handler {
	h := Handler{
		ctx:                scaled.Ctx,
		knowledgeBaseCache: scaled.Management.AgentFactory.Agent().V1().KnowledgeBase().Cache(),
	}

	return h
}

func (h Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if err := h.do(rw, req); err != nil {
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

	switch action {
	case ActionSearch:
		return h.search(rw, req, namespace, name)
	case ActionListObjects:
		return h.listObjects(rw, req, namespace, name)
	default:
		return apierror.NewAPIError(validation.InvalidAction, fmt.Sprintf("Unsupported POST action %s", action))
	}
}

func (h Handler) search(rw http.ResponseWriter, req *http.Request, namespace, name string) error {
	var input SearchInput
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		return apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("Failed to parse body: %v", err))
	}

	kb, err := h.knowledgeBaseCache.Get(namespace, name)
	if err != nil {
		return apierror.NewAPIError(validation.NotFound, fmt.Sprintf("Failed to get knownledgebase %s/%s: %v", namespace, name, err))
	}

	c, err := helper.NewVectorDatabaseClient(kb.Spec.EmbeddingModel)
	if err != nil {
		return fmt.Errorf("failed to create vector database client: %w", err)
	}

	// Convert name to valid Weaviate class name
	result, err := c.Search(h.ctx, kb.Status.ClassName, input.Query, input.Threshold, input.Limit)
	if err != nil {
		return fmt.Errorf("failed to search %s with query string %s: %w", kb.Status.ClassName, input.Query, err)
	}

	utils.ResponseOKWithBody(rw, result)

	return nil
}

func (h Handler) listObjects(rw http.ResponseWriter, req *http.Request, namespace, name string) error {
	var input ListObjectsInput
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		return apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("Failed to parse body: %v", err))
	}

	kb, err := h.knowledgeBaseCache.Get(namespace, name)
	if err != nil {
		return apierror.NewAPIError(validation.NotFound, fmt.Sprintf("Failed to get knownledgebase %s/%s: %v", namespace, name, err))
	}

	c, err := helper.NewVectorDatabaseClient(kb.Spec.EmbeddingModel)
	if err != nil {
		return fmt.Errorf("failed to create vector database client: %w", err)
	}

	objects, err := c.ListObjects(h.ctx, kb.Status.ClassName, input.UID, input.Offset, input.Limit)
	if err != nil {
		return fmt.Errorf("failed to list objects with class name %s: %w", kb.Status.ClassName, err)
	}

	utils.ResponseOKWithBody(rw, objects)

	return nil
}
