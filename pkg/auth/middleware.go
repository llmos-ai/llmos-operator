package auth

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	authUser "k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/endpoints/request"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/auth/tokens"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/indexeres"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

var (
	ErrMustAuthenticate = errors.New("must authenticate")
	ErrMsgEmptyList     = errors.New("empty list")
)

func NewMiddleware(scaled *config.Scaled) *Middleware {
	users := scaled.MgmtFactory.Management().V1().User()
	tokens := scaled.MgmtFactory.Management().V1().Token()
	return &Middleware{
		userClient:  users,
		userCache:   users.Cache(),
		tokenClient: tokens,
		tokenCache:  tokens.Cache(),
	}
}

type Middleware struct {
	userClient  ctlmgmtv1.UserClient
	userCache   ctlmgmtv1.UserCache
	tokenClient ctlmgmtv1.TokenClient
	tokenCache  ctlmgmtv1.TokenCache
}

func (m *Middleware) AuthMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		token := tokens.ExtractTokenFromRequest(req)
		userInfo, err := m.GetUserInfoFromToken(token)
		if err != nil {
			utils.ResponseError(rw, http.StatusUnauthorized, err)
			return
		}

		ctx := request.WithUser(req.Context(), userInfo)
		req = req.WithContext(ctx)
		handler.ServeHTTP(rw, req)
	})
}

func (m *Middleware) GetUserInfoFromToken(tokenStr string) (authUser.Info, error) {
	token, err := m.GetTokenFromRequest(tokenStr)
	if err != nil {
		return nil, err
	}

	user, err := m.GetUserByName(token.Spec.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user %s, %v", token.Spec.UserId, err)
	}

	if !user.Status.IsActive {
		return nil, errors.Wrap(ErrMustAuthenticate, "user is not enabled")
	}

	var userInfo authUser.DefaultInfo
	userInfo.Name = user.Name
	userInfo.UID = string(user.UID)
	userInfo.Groups = []string{
		"system:authenticated",
	}
	if user.Status.IsAdmin {
		userInfo.Groups = append(userInfo.Groups, constant.AdminRole)
	}

	return &userInfo, nil
}

func (m *Middleware) GetTokenFromRequest(tokenAuthValue string) (*mgmtv1.Token, error) {
	tokenName, tokenKey := tokens.SplitTokenParts(tokenAuthValue)
	if tokenName == "" || tokenKey == "" {
		return nil, ErrMustAuthenticate
	}

	usingClient := false
	token, err := m.TokenIndexByTokenKey(tokenName)
	if err != nil {
		if errors.Is(err, ErrMsgEmptyList) {
			usingClient = true
		} else {
			return nil, err
		}
	}

	if usingClient {
		token, err = m.tokenClient.Get(tokenName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, ErrMustAuthenticate
			}
			return nil, errors.Wrapf(ErrMustAuthenticate, "failed to get auth token, error: %v", err)
		}
	}

	if _, err = tokens.VerifyToken(token, tokenName, tokenKey); err != nil {
		return nil, errors.Wrapf(ErrMustAuthenticate, "failed to verify token: %v", err)
	}

	return token, nil
}

func (m *Middleware) GetUserByName(username string) (*mgmtv1.User, error) {
	usingClient := false
	user, err := m.UserIndexByUserName(username)
	if err != nil {
		if errors.Is(err, ErrMsgEmptyList) {
			usingClient = true
		} else {
			return nil, err
		}
	}

	if usingClient {
		user, err = m.userClient.Get(username, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, ErrMustAuthenticate
			}
			return nil, errors.Wrapf(ErrMustAuthenticate, "failed to get user, error: %v", err)
		}
	}

	return user, nil
}
func (m *Middleware) UserIndexByUserName(name string) (*mgmtv1.User, error) {
	objs, err := m.userCache.GetByIndex(indexeres.UserNameIndex, name)
	if err != nil {
		return nil, err
	}
	if len(objs) == 0 {
		return nil, ErrMsgEmptyList
	}
	if len(objs) > 1 {
		return nil, errors.New("found more than one users with name " + name)
	}
	return objs[0], nil
}

func (m *Middleware) TokenIndexByTokenKey(tokenKey string) (*mgmtv1.Token, error) {
	objs, err := m.tokenCache.GetByIndex(indexeres.TokenNameIndex, tokenKey)
	if err != nil {
		return nil, err
	}
	if len(objs) == 0 {
		return nil, ErrMsgEmptyList
	}
	if len(objs) > 1 {
		return nil, errors.New("found more than one tokens")
	}
	return objs[0], nil
}
