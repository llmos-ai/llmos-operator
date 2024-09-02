package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/sirupsen/logrus"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/auth"
	"github.com/llmos-ai/llmos-operator/pkg/auth/tokens"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

const (
	actionQuery      = "action"
	loginActionName  = "login"
	logoutActionName = "logout"

	UserNotActiveErrMsg = "User is not activated"
)

type LoginRequest struct {
	Username     string `json:"username" binding:"required"`
	Password     string `json:"password" binding:"required"`
	ResponseType string `json:"responseType"`
}

type LoginResponse struct {
	UserId       string `json:"userId"`
	AuthProvider string `json:"authProvider"`
	Token        string `json:"token"`
}

type Handler struct {
	manager    *tokens.Manager
	middleware *auth.Middleware
}

func NewAuthHandler(mgmt *config.Management) *Handler {
	middleware := auth.NewMiddleware(mgmt)
	manager := tokens.NewManager(mgmt)
	return &Handler{
		manager:    manager,
		middleware: middleware,
	}
}

func (h *Handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponseError(rw, http.StatusMethodNotAllowed, constant.ErrOnlyPostMethod)
		return
	}

	action := strings.ToLower(r.URL.Query().Get(actionQuery))
	switch action {
	case logoutActionName:
		err := h.manager.Logout(r, rw)
		if err != nil {
			utils.ResponseError(rw, http.StatusInternalServerError, err)
			return
		}
		utils.ResponseOKWithBody(rw, []byte("success logout"))
		return
	case loginActionName:
		var input LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			utils.ResponseError(rw, http.StatusBadRequest, fmt.Errorf("failed to decode request body, %s", err.Error()))
			return
		}

		userId, tokenResp, err := h.login(&input)
		if err != nil {
			header := http.StatusInternalServerError
			var e *apierror.APIError
			if errors.As(err, &e) {
				header = e.Code.Status
			}
			utils.ResponseErrorMsg(rw, header, e.Message)
			return
		}

		if input.ResponseType == "cookie" {
			tokenCookie := &http.Cookie{
				Name:     tokens.CookieName,
				Value:    tokenResp,
				Path:     "/",
				Secure:   true,
				HttpOnly: true,
			}

			http.SetCookie(rw, tokenCookie)
			utils.ResponseOKWithBody(rw, "login success")
			return
		}

		utils.ResponseOKWithBody(rw, &LoginResponse{
			UserId:       userId,
			AuthProvider: tokens.LocalProviderName,
			Token:        tokenResp,
		})
		return
	default:
		rw.WriteHeader(http.StatusBadRequest)
		utils.ResponseError(rw, http.StatusBadRequest, constant.ErrUnsupportedAction)
		return
	}
}

func (h *Handler) login(input *LoginRequest) (string, string, error) {
	user, err := h.userLogin(input)
	if err != nil {
		return "", "", err
	}

	token, err := h.generateToken(user.Name)
	if err != nil {
		return "", "", apierror.NewAPIError(validation.ServerError,
			fmt.Sprintf("failed to generate token, %s", err.Error()))
	}

	return user.Name, token, nil
}

func (h *Handler) userLogin(input *LoginRequest) (*mgmtv1.User, error) {
	username := input.Username
	pwd := input.Password

	user, err := h.middleware.GetUserByName(username)
	if err != nil {
		logrus.Debugf("failed to get user %s, %s", username, err.Error())
		return nil, apierror.NewAPIError(validation.Unauthorized, "authentication failed")
	}

	if err = checkUserIsActive(user); err != nil {
		return nil, err
	}

	if !tokens.CheckPasswordHash(user.Spec.Password, pwd) {
		return nil, apierror.NewAPIError(validation.Unauthorized, "authentication failed")
	}

	return user, nil
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
func (h *Handler) generateToken(userId string) (string, error) {
	authTimeout := settings.AuthUserSessionMaxTTLMinutes.Get()
	ttl, err := strconv.ParseInt(authTimeout, 10, 64)
	if err != nil {
		logrus.Errorf("failed to parse auth-user-session-max-ttl, use default 12hrs, %s", err.Error())
		ttl = 720
	}

	token, tokenStr, err := h.manager.NewLoginToken(userId, ttl*60) // convert ttl to seconds
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", token.Name, tokenStr), nil
}
