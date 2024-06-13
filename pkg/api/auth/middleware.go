package auth

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	authUser "k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"

	auth2 "github.com/llmos-ai/llmos-controller/pkg/auth"
	"github.com/llmos-ai/llmos-controller/pkg/constant"
	"github.com/llmos-ai/llmos-controller/pkg/server/config"
	"github.com/llmos-ai/llmos-controller/pkg/utils"
)

func NewMiddleware(mgmt *config.Management) *Middleware {
	handler := NewAuthHandler(mgmt)
	return &Middleware{
		mgmt:    mgmt,
		handler: handler,
	}
}

type Middleware struct {
	mgmt    *config.Management
	handler *Handler
}

func (m *Middleware) AuthMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		jweToken, err := extractJWTTokenFromRequest(req)
		if err != nil {
			utils.ResponseError(rw, http.StatusUnauthorized, err)
			return
		}

		userInfo, err := m.getUserInfoFromToken(jweToken)
		if err != nil {
			utils.ResponseError(rw, http.StatusUnauthorized, err)
			return
		}

		ctx := request.WithUser(req.Context(), userInfo)
		req = req.WithContext(ctx)
		handler.ServeHTTP(rw, req)
	})
}

func extractJWTTokenFromRequest(req *http.Request) (string, error) {
	tokenStr := req.Header.Get("Authorization")
	if strings.HasPrefix(tokenStr, "Bearer ") {
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	} else {
		tokenStr = ""
	}

	if tokenStr == "" {
		cookie, err := req.Cookie(CookieName)
		if err != nil && !errors.Is(err, http.ErrNoCookie) {
			return tokenStr, err
		} else if !errors.Is(err, http.ErrNoCookie) && len(cookie.Value) > 0 {
			tokenStr = cookie.Value
		}
	}

	if tokenStr == "" {
		return "", errors.New("failed to get cookie from request")
	}

	decodedToken, err := url.QueryUnescape(tokenStr)
	if err != nil {
		return "", errors.New("failed to parse cookie from request")
	}
	return decodedToken, nil
}

func (m *Middleware) getUserInfoFromToken(tokenStr string) (authUser.Info, error) {
	claims, err := auth2.VerifyToken(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	user, err := m.handler.getUserByUID(string(claims.UID))
	if err != nil {
		return nil, err
	}

	var userInfo authUser.DefaultInfo
	if user.Username != "" {
		userInfo.Name = user.Name
		userInfo.UID = string(user.UID)
		userInfo.Groups = []string{
			"system:authenticated",
		}
		if user.IsAdmin {
			userInfo.Groups = append(userInfo.Groups, constant.AdminRole)
		}
	}

	return &userInfo, nil
}
