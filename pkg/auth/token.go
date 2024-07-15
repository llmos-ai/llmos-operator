package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/types"

	"github.com/llmos-ai/llmos-operator/pkg/settings"
)

type Claims struct {
	UID types.UID
	jwt.RegisteredClaims
}

const (
	hashCost = 10
)

func GenerateToken(uid types.UID) (string, error) {
	duration, err := time.ParseDuration(fmt.Sprintf("%sm", settings.AuthTokenMaxTTLMinutes.Get()))
	if err != nil {
		return "", err
	}
	time := time.Now().Add(duration)

	claims := Claims{
		UID: uid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time),
			Issuer:    settings.AuthSecretName.Get(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString([]byte(settings.AuthSecretName.Get()))
	if err != nil {
		return "", err
	}

	return ss, nil
}

func VerifyToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(settings.AuthSecretName.Get()), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	return string(bytes), err
}

func CheckPasswordHash(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
