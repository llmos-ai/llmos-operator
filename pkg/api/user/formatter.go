package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/llmos-ai/llmos-operator/pkg/auth"
	"github.com/llmos-ai/llmos-operator/pkg/auth/tokens"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

func Formatter(request *types.APIRequest, resource *types.RawResource) {
	resource.Actions = make(map[string]string, 1)
	if request.AccessControl.CanUpdate(request, resource.APIObject, resource.Schema) != nil {
		return
	}
	resource.AddAction(request, ActionSetIsActive)
}

func CollectionFormatter(request *types.APIRequest, collection *types.GenericCollection) {
	collection.AddAction(request, ActionChangePassword)
}

type Handler struct {
	userClient ctlmgmtv1.UserClient
	userCache  ctlmgmtv1.UserCache
	middleware *auth.Middleware
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

func (h Handler) doPost(_ http.ResponseWriter, req *http.Request, vars map[string]string) error {
	action := vars["action"]
	name := vars["name"]
	switch action {
	case ActionSetIsActive:
		return h.setIsActive(name, req)
	case ActionChangePassword:
		return h.changeCurrentUserPassword(req)
	default:
		return apierror.NewAPIError(validation.InvalidAction, fmt.Sprintf("Unsupported POST action %s", action))
	}
}

func (h Handler) setIsActive(name string, req *http.Request) error {
	input := &SetIsActiveInput{}
	if err := json.NewDecoder(req.Body).Decode(input); err != nil {
		return apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("Failed to parse body: %v", err))
	}

	// check if user exists
	user, err := h.userCache.Get(name)
	if err != nil {
		return err
	}

	userCpy := user.DeepCopy()
	userCpy.Spec.Active = input.IsActive
	if _, err = h.userClient.Update(userCpy); err != nil {
		return err
	}
	return nil
}

func (h Handler) userListHandler(request *types.APIRequest) (types.APIObjectList, error) {
	if request.Name == "" {
		if err := request.AccessControl.CanList(request, request.Schema); err != nil {
			return types.APIObjectList{}, err
		}
	} else {
		if err := request.AccessControl.CanGet(request, request.Schema); err != nil {
			return types.APIObjectList{}, err
		}
	}

	store := request.Schema.Store
	if store == nil {
		return types.APIObjectList{}, apierror.NewAPIError(validation.NotFound, "no store found")
	}

	query := request.Query
	me := query.Get("me")
	if me == "true" {
		user := request.GetUser()
		userObj, err := store.ByID(request, request.Schema, user)
		if err != nil {
			return types.APIObjectList{}, err
		}
		return types.APIObjectList{
			Objects: []types.APIObject{userObj},
		}, nil
	}

	return store.List(request, request.Schema)
}

func (h Handler) changeCurrentUserPassword(req *http.Request) error {
	input := &ChangePasswordInput{}
	if err := json.NewDecoder(req.Body).Decode(input); err != nil {
		return apierror.NewAPIError(validation.InvalidBodyContent, fmt.Sprintf("Failed to parse body: %v", err))
	}

	userInfo, authed := request.UserFrom(req.Context())
	if !authed {
		return apierror.NewAPIError(validation.Unauthorized, "Unauthorized")
	}

	user, err := h.userCache.Get(userInfo.GetName())
	if err != nil {
		return apierror.NewAPIError(validation.InvalidAction, fmt.Sprintf("failed to get user: %v", err))
	}

	if valid := tokens.CheckPasswordHash(user.Spec.Password, input.CurrentPassword); !valid {
		return apierror.NewAPIError(validation.InvalidBodyContent, "Current password is incorrect")
	}

	toUpdate := user.DeepCopy()
	toUpdate.Spec.Password = input.NewPassword
	if _, err = h.userClient.Update(toUpdate); err != nil {
		return apierror.NewAPIError(validation.ServerError, err.Error())
	}

	return nil
}
