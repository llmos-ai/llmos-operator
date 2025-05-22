package tokens

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

type tokenKind string

const (
	LabelAuthUserId    = "auth.management.llmos.ai/user-id"
	LabelAuthTokenKind = "auth.management.llmos.ai/kind"

	LocalProviderName = "local"
	CookieName        = "L_SESS"
	CSRFCookie        = "CSRF"

	sessionToken tokenKind = "ui-session"
	apiKeyToken  tokenKind = "api-key"
)

var (
	ErrInvalidTokenFormat = errors.New("invalid token format")

	toDeleteCookies = []string{CookieName, CSRFCookie}
)

type Manager struct {
	tokensClient ctlmgmtv1.TokenClient
}

func NewManager(scaled *config.Scaled) *Manager {
	tokens := scaled.MgmtFactory.Management().V1().Token()

	return &Manager{
		tokensClient: tokens,
	}
}

func (m *Manager) NewLoginToken(userId string, ttl int64) (*mgmtv1.Token, string, error) {
	return m.createToken("token-", userId, nil, sessionToken, ttl)
}

func (m *Manager) NewAPIKeyToken(userId string, ttl int64, token *mgmtv1.Token) (*mgmtv1.Token, string, error) {
	return m.createToken("llmos-", userId, token, apiKeyToken, ttl)
}

func (m *Manager) createToken(generateName, userId string, token *mgmtv1.Token, kind tokenKind,
	ttl int64) (*mgmtv1.Token, string, error) {
	key, err := utils.GenerateToken()
	if err != nil {
		logrus.Errorf("Failed to generate token key: %v", err)
		return nil, "", fmt.Errorf("failed to generate token key")
	}

	hashedToken, err := ConvertTokenKeyToHash(key)
	if err != nil {
		return nil, "", err
	}

	toCreate := &mgmtv1.Token{}
	if token != nil {
		toCreate = token.DeepCopy()
		if toCreate.Labels == nil {
			toCreate.Labels = map[string]string{}
		}
		toCreate.Labels[LabelAuthUserId] = userId
		toCreate.Labels[LabelAuthTokenKind] = string(kind)
		toCreate.Spec = mgmtv1.TokenSpec{
			AuthProvider: LocalProviderName,
			Expired:      ttl != 0,
			UserId:       userId,
			TTLSeconds:   ttl,
			Token:        hashedToken,
		}
	} else {
		toCreate = &mgmtv1.Token{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: generateName,
				Labels: map[string]string{
					LabelAuthUserId:    userId,
					LabelAuthTokenKind: string(kind),
				},
				Annotations: map[string]string{},
			},
			Spec: mgmtv1.TokenSpec{
				AuthProvider: LocalProviderName,
				Expired:      ttl != 0,
				UserId:       userId,
				TTLSeconds:   ttl,
				Token:        hashedToken,
			},
		}
	}

	token, err = m.tokensClient.Create(toCreate)
	if err != nil {
		return nil, "", err
	}
	return token, key, nil
}

func (m *Manager) Logout(req *http.Request, rw http.ResponseWriter) error {
	tokenAuthValue := ExtractTokenFromRequest(req)

	tokenName, tokenKey := SplitTokenParts(tokenAuthValue)
	if tokenName == "" || tokenKey == "" {
		return ErrInvalidTokenFormat
	}

	isSecure := req.URL.Scheme == "https"

	for _, cookieName := range toDeleteCookies {
		tokenCookie := &http.Cookie{
			Name:     cookieName,
			Value:    "",
			Secure:   isSecure,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
			Expires:  time.Unix(1, 0), //January 1, 1970 UTC
		}
		http.SetCookie(rw, tokenCookie)
	}

	err := m.deleteTokenByName(tokenName)
	if err != nil {
		return fmt.Errorf("failed to delete token, err %s", err.Error())
	}
	return nil
}

func (m *Manager) deleteTokenByName(tokenName string) error {
	err := m.tokensClient.Delete(tokenName, &metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete token, err %s", err.Error())
	}
	logrus.Debugf("Deleted token %s", tokenName)
	return nil
}
