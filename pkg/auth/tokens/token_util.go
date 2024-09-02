package tokens

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/auth/hashers"
)

const (
	hashCost        = 10
	AuthHeaderName  = "Authorization"
	AuthValuePrefix = "Bearer"
	BasicAuthPrefix = "Basic"
)

func SplitTokenParts(tokenID string) (string, string) {
	parts := strings.Split(tokenID, ":")
	if len(parts) != 2 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func SetTokenExpiresAt(token *mgmtv1.Token) {
	if token.Spec.TTLSeconds != 0 {
		created := token.ObjectMeta.CreationTimestamp.Time
		ttlDuration := time.Duration(token.Spec.TTLSeconds) * time.Second
		expiresAtTime := created.Add(ttlDuration)
		token.Status.ExpiresAt = metav1.NewTime(expiresAtTime)
	}
}

func IsExpired(token *mgmtv1.Token) bool {
	if token.Spec.TTLSeconds == 0 {
		return false
	}

	created := token.ObjectMeta.CreationTimestamp.Time
	durationElapsed := time.Since(created)

	ttlDuration := time.Duration(token.Spec.TTLSeconds) * time.Second
	return durationElapsed.Seconds() >= ttlDuration.Seconds()
}

// VerifyToken helps to check if the token is valid
func VerifyToken(token *mgmtv1.Token, tokenName, tokenKey string) (int, error) {
	invalidAuthTokenErr := errors.New("invalid auth token value")

	if token == nil || token.Name != tokenName {
		return http.StatusUnprocessableEntity, invalidAuthTokenErr
	}
	hasher := hashers.NewHasher()
	if err := hasher.VerifyHash(token.Spec.Token, tokenKey); err != nil {
		logrus.Errorf("VerifyHash failed with error: %v", err)
		return http.StatusUnprocessableEntity, invalidAuthTokenErr
	}

	if IsExpired(token) {
		return http.StatusUnauthorized, errors.New("token expired")
	}
	return http.StatusOK, nil
}

// ConvertTokenKeyToHash helps to convert token key to hash
func ConvertTokenKeyToHash(token string) (string, error) {
	hash := hashers.NewHasher()
	hashedToken, err := hash.CreateHash(token)
	if err != nil {
		return "", fmt.Errorf("failed to generate hash from token: %v", err)
	}
	return hashedToken, nil
}
func ExtractTokenFromRequest(req *http.Request) string {
	var tokenAuthValue string
	authHeader := req.Header.Get(AuthHeaderName)
	authHeader = strings.TrimSpace(authHeader)

	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if strings.EqualFold(parts[0], AuthValuePrefix) {
			if len(parts) > 1 {
				tokenAuthValue = strings.TrimSpace(parts[1])
			}
		} else if strings.EqualFold(parts[0], BasicAuthPrefix) {
			if len(parts) > 1 {
				base64Value := strings.TrimSpace(parts[1])
				data, err := base64.URLEncoding.DecodeString(base64Value)
				if err != nil {
					logrus.Errorf("Error %v parsing %v header", err, AuthHeaderName)
				} else {
					tokenAuthValue = string(data)
				}
			}
		}
	} else {
		cookie, err := req.Cookie(CookieName)
		if err == nil {
			tokenAuthValue = cookie.Value
		}
	}
	return tokenAuthValue
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	return string(bytes), err
}

func CheckPasswordHash(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
