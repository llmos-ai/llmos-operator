package user

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/wrangler/v2/pkg/schemas/validation"

	"github.com/llmos-ai/llmos-controller/pkg/utils"
)

func formatter(request *types.APIRequest, resource *types.RawResource) {
	resource.Actions = make(map[string]string, 1)
	if request.AccessControl.CanUpdate(request, resource.APIObject, resource.Schema) != nil {
		return
	}
	resource.AddAction(request, ActionSetIsActive)
}

func (h Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if err := h.do(rw, req); err != nil {
		status := http.StatusInternalServerError
		var e *apierror.APIError
		if errors.As(err, &e) {
			status = e.Code.Status
		}
		rw.WriteHeader(status)
		_, _ = rw.Write([]byte(err.Error()))
		return
	}
	rw.WriteHeader(http.StatusNoContent)
}

func (h Handler) do(rw http.ResponseWriter, req *http.Request) error {
	vars := utils.EncodeVars(mux.Vars(req))
	if req.Method == http.MethodPost {
		return h.doPost(vars["action"], rw, req)
	}

	return apierror.NewAPIError(validation.InvalidAction, fmt.Sprintf("Unsupported method %s", req.Method))
}

func (h Handler) doPost(action string, rw http.ResponseWriter, req *http.Request) error {
	vars := utils.EncodeVars(mux.Vars(req))
	name := vars["name"]
	switch action {
	case ActionSetIsActive:
		return h.setIsActive(name)
	default:
		return apierror.NewAPIError(validation.InvalidAction, fmt.Sprintf("Unsupported POST action %s", action))
	}
}

func (h Handler) setIsActive(name string) error {
	// check if user exists
	user, err := h.userCache.Get(name)
	if err != nil {
		return err
	}
	userCpy := user.DeepCopy()
	userCpy.Spec.IsActive = !user.Spec.IsActive
	if _, err = h.user.Update(userCpy); err != nil {
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
