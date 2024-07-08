package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/wrangler/v2/pkg/schemas/validation"

	mgmtv1 "github.com/llmos-ai/llmos-controller/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-controller/pkg/auth"
	ctlmgmtv1 "github.com/llmos-ai/llmos-controller/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-controller/pkg/indexeres"
	"github.com/llmos-ai/llmos-controller/pkg/server/config"
	"github.com/llmos-ai/llmos-controller/pkg/utils"
)

const (
	CookieName       = "L_SESS"
	actionQuery      = "action"
	loginActionName  = "login"
	logoutActionName = "logout"

	UserNotActiveErrMsg = "user is not active"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type Handler struct {
	user      ctlmgmtv1.UserClient
	userCache ctlmgmtv1.UserCache
}

func NewAuthHandler(mgmt *config.Management) *Handler {
	users := mgmt.MgmtFactory.Management().V1().User()
	return &Handler{
		user:      users,
		userCache: users.Cache(),
	}
}

func (h *Handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponseError(rw, http.StatusMethodNotAllowed, fmt.Errorf("only POST method is supported"))
		return
	}

	action := strings.ToLower(r.URL.Query().Get(actionQuery))
	if action == logoutActionName {
		// erase the cookie
		tokenCookie := &http.Cookie{
			Name:    CookieName,
			Value:   "",
			Path:    "/",
			MaxAge:  -1,
			Expires: time.Unix(1, 0), //January 1, 1970 UTC
		}
		http.SetCookie(rw, tokenCookie)
		utils.ResponseOKWithBody(rw, []byte("success logout"))
		return
	}

	if action != loginActionName {
		rw.WriteHeader(http.StatusBadRequest)
		utils.ResponseError(rw, http.StatusBadRequest, fmt.Errorf("unsupported action"))
		return
	}

	var input LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.ResponseError(rw, http.StatusBadRequest, fmt.Errorf("failed to decode request body, %s", err.Error()))
		return
	}

	tokenResp, err := h.login(&input)
	var header int
	if err != nil {
		var e *apierror.APIError
		if errors.As(err, &e) {
			header = e.Code.Status
		} else {
			header = http.StatusInternalServerError
		}
		utils.ResponseError(rw, header, err)
		return
	}

	tokenCookie := &http.Cookie{
		Name:  CookieName,
		Value: tokenResp,
		Path:  "/",
	}

	http.SetCookie(rw, tokenCookie)
	rw.Header().Set("Content-type", "application/json")
	utils.ResponseOKWithBody(rw, "login success")
}

func (h *Handler) login(input *LoginRequest) (token string, err error) {
	user, err := h.userLogin(input)
	if err != nil {
		return "", err
	}

	token, err = auth.GenerateToken(user.UID)
	if err != nil {
		return "", fmt.Errorf("failed to generate token, %s", err.Error())
	}

	escapedToken := url.QueryEscape(token)
	return escapedToken, nil
}

func (h *Handler) userLogin(input *LoginRequest) (*mgmtv1.User, error) {
	username := input.Username
	pwd := input.Password

	user, err := h.getUser(username)
	if err != nil {
		return nil, apierror.NewAPIError(validation.Unauthorized, err.Error())
	}

	if err = checkUserIsActive(user); err != nil {
		return nil, err
	}

	if !auth.CheckPasswordHash(user.Spec.Password, pwd) {
		return nil, apierror.NewAPIError(validation.Unauthorized, "authentication failed")
	}

	return user, nil
}

func (h *Handler) getUser(username string) (*mgmtv1.User, error) {
	objs, err := h.userCache.GetByIndex(indexeres.UserNameIndex, username)
	if err != nil {
		return nil, err
	}
	if len(objs) == 0 {
		return nil, errors.New("authentication failed")
	}
	if len(objs) > 1 {
		return nil, errors.New("found more than one users with username " + username)
	}
	return objs[0], nil
}

func (h *Handler) getUserByUID(uid string) (*mgmtv1.User, error) {
	objs, err := h.userCache.GetByIndex(indexeres.UserUIDIndex, uid)
	if err != nil {
		return nil, err
	}
	if len(objs) == 0 {
		return nil, errors.New("authentication failed")
	}
	if len(objs) > 1 {
		return nil, errors.New("found more than one users with uid " + uid)
	}
	return objs[0], nil
}

func checkUserIsActive(user *mgmtv1.User) error {
	if user == nil {
		return apierror.NewAPIError(validation.Unauthorized, "user object is nil")
	}

	if !user.Status.IsActive {
		return apierror.NewAPIError(validation.Unauthorized, UserNotActiveErrMsg)
	}

	return nil
}
