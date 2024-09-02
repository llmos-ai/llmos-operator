package token

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/steve/pkg/schema"
	"github.com/rancher/steve/pkg/server"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/auth"
	tokens2 "github.com/llmos-ai/llmos-operator/pkg/auth/tokens"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/settings"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

const (
	tokenSchemaID = "management.llmos.ai.token"
)

type handler struct {
	httpClient  http.Client
	tokens      ctlmgmtv1.TokenClient
	tokensCache ctlmgmtv1.TokenCache
	manager     *tokens2.Manager
	middleware  *auth.Middleware
}

func formatter(_ *types.APIRequest, resource *types.RawResource) {
	resource.Actions = nil
	delete(resource.Links, "update")
}

func RegisterSchema(mgmt *config.Management, server *server.Server) error {
	tokens := mgmt.MgmtFactory.Management().V1().Token()
	h := &handler{
		httpClient:  http.Client{},
		tokens:      tokens,
		tokensCache: tokens.Cache(),
		manager:     tokens2.NewManager(mgmt),
		middleware:  auth.NewMiddleware(mgmt),
	}

	t := []schema.Template{
		{
			ID:        tokenSchemaID,
			Formatter: formatter,
			Customize: func(apiSchema *types.APISchema) {
				apiSchema.CreateHandler = h.createHandler
				apiSchema.ListHandler = h.listHandler
			},
		},
	}

	server.SchemaFactory.AddTemplate(t...)
	return nil
}
func (h *handler) createHandler(request *types.APIRequest) (types.APIObject, error) {
	r := request.Request
	rw := request.Response

	var token = &mgmtv1.Token{}
	err := json.NewDecoder(r.Body).Decode(token)
	if err != nil {
		utils.ResponseError(rw, http.StatusBadRequest, err)
		return types.APIObject{}, fmt.Errorf("error decoding request body: %v", err)
	}

	token, originTokenStr, err := h.constructAPIKeyToken(r, token)
	if err != nil {
		utils.ResponseError(rw, http.StatusBadRequest, err)
		return types.APIObject{}, fmt.Errorf("error generating token: %v", err)
	}

	token.Spec.Token = originTokenStr

	return types.APIObject{
		ID:     token.Name,
		Type:   tokenSchemaID,
		Object: token,
	}, nil
}

func (h *handler) constructAPIKeyToken(req *http.Request, token *mgmtv1.Token) (*mgmtv1.Token, string, error) {
	user, _, err := h.getUserBySessionToken(req)
	if err != nil {
		return nil, "", err
	}

	return h.generateToken(user.Name, token)
}

func (h *handler) getUserBySessionToken(r *http.Request) (*mgmtv1.User, *mgmtv1.Token, error) {
	tokenStr := tokens2.ExtractTokenFromRequest(r)

	sessionToken, err := h.middleware.GetTokenFromRequest(tokenStr)
	if err != nil {
		return nil, nil, err
	}

	user, err := h.middleware.GetUserByName(sessionToken.Spec.UserId)
	if err != nil {
		return nil, nil, err
	}

	if !user.Status.IsActive {
		return nil, nil, fmt.Errorf("user is not activated")
	}

	return user, sessionToken, nil
}

func (h *handler) generateToken(userId string, token *mgmtv1.Token) (*mgmtv1.Token, string, error) {
	authTimeout := settings.AuthTokenMaxTTLMinutes.Get()
	maxTTL, err := strconv.ParseInt(authTimeout, 10, 64)
	if err != nil {
		logrus.Errorf("failed to parse auth-user-session-max-maxTTL, use default 90 days, %s", err.Error())
		maxTTL = 129600
	}

	// user input ttl should not be greater than system's max maxTTL in seconds
	if token.Spec.TTLSeconds <= maxTTL*60 {
		maxTTL = token.Spec.TTLSeconds
	}

	token, tokenStr, err := h.manager.NewAPIKeyToken(userId, maxTTL, token)
	if err != nil {
		return nil, "", err
	}

	return token, fmt.Sprintf("%s:%s", token.Name, tokenStr), nil
}

func (h *handler) listHandler(request *types.APIRequest) (types.APIObjectList, error) {
	currentUser, sessionToken, err := h.getUserBySessionToken(request.Request)
	if err != nil {
		utils.ResponseError(request.Response, http.StatusBadRequest, err)
		return types.APIObjectList{}, err
	}
	selector := labels.SelectorFromSet(map[string]string{
		tokens2.UserIDLabel: currentUser.Name,
	})

	tokens, err := h.tokensCache.List(selector)
	if err != nil {
		utils.ResponseError(request.Response, http.StatusBadRequest, err)
		return types.APIObjectList{}, err
	}

	// convert token into api object
	result := ConvertTokenListToAPIObjectList(tokens, sessionToken)

	return types.APIObjectList{
		Objects: result,
	}, nil
}
