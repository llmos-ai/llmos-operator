package tokens

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
)

func TestVerifyToken(t *testing.T) {
	tokenName := "test-token"

	tokenKey := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	badTokenKey := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa@"

	// SHA3 hash of tokenKey
	hashedTokenKey := "$1:YJOvkxN/BqY:D6xNHoUYM4wiNDJHYUGkqO2GZPAvNy11QdiyGQSyStORuIyK7JTwdapfwPOw3WpFp15T2JlLCxC6rNKubs1mag" //nolint:lll
	invalidHashToken := "$-1:111:111"
	hashedToken := mgmtv1.Token{
		ObjectMeta: metav1.ObjectMeta{
			Name: tokenName,
		},
		Spec: mgmtv1.TokenSpec{
			Token:      hashedTokenKey,
			TTLSeconds: 0,
		},
	}
	// valid hashed token with bad name
	wrongToken := *hashedToken.DeepCopy()
	wrongToken.Name = "wrong-token"

	// invalid hashed token
	invalidHashedToken := *hashedToken.DeepCopy()
	invalidHashedToken.Spec.Token = invalidHashToken

	tests := []struct {
		name      string
		token     *mgmtv1.Token
		tokenName string
		tokenKey  string

		wantResponseCode int
		wantErr          bool
	}{
		{
			name:             "valid hashed token",
			token:            &hashedToken,
			tokenName:        tokenName,
			tokenKey:         tokenKey,
			wantResponseCode: 200,
		},
		{
			name:             "valid hashed token, incorrect key",
			token:            &hashedToken,
			tokenName:        tokenName,
			tokenKey:         badTokenKey,
			wantResponseCode: 422,
			wantErr:          true,
		},
		{
			name:             "wrong token",
			token:            &wrongToken,
			tokenName:        tokenName,
			tokenKey:         tokenKey,
			wantResponseCode: 422,
			wantErr:          true,
		},
		{
			name:             "incorrect token key",
			token:            &hashedToken,
			tokenName:        tokenName,
			tokenKey:         badTokenKey,
			wantResponseCode: 422,
			wantErr:          true,
		},
		{
			name:             "expired token",
			token:            expireToken(&hashedToken),
			tokenName:        tokenName,
			tokenKey:         tokenKey,
			wantResponseCode: 401,
			wantErr:          true,
		},
		{
			name:             "nil token",
			token:            nil,
			tokenName:        tokenName,
			tokenKey:         tokenKey,
			wantResponseCode: 422,
			wantErr:          true,
		},
		{
			name:             "invalid hash token",
			token:            &invalidHashedToken,
			tokenName:        tokenName,
			tokenKey:         tokenKey,
			wantResponseCode: 422,
			wantErr:          true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			responseCode, err := VerifyToken(test.token, test.tokenName, test.tokenKey)
			if test.wantErr {
				require.Error(t, err)
			}
			require.Equal(t, test.wantResponseCode, responseCode)
		})
	}
}

func TestConvertTokenKeyToHash(t *testing.T) {
	secretKey := "ccccccccccccccccccccccccccccccccccccc"
	fancySecretKey := "@#$%^&*()_+_)(*&^%$#@#$%^&*()"
	tests := []struct {
		name       string
		secretKey  string
		wantedHash bool
		wantError  bool
	}{
		{
			name:       "valid token hashing",
			secretKey:  secretKey,
			wantedHash: true,
			wantError:  false,
		},
		{
			name:       "fancy key token hashing",
			secretKey:  fancySecretKey,
			wantedHash: true,
			wantError:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hash, err := ConvertTokenKeyToHash(test.secretKey)
			if test.wantError {
				require.Error(t, err)
			}
			require.Nil(t, err)

			if test.wantedHash {
				require.NotEmpty(t, hash)
			}
		})
	}
}

func expireToken(token *mgmtv1.Token) *mgmtv1.Token {
	newToken := token.DeepCopy()
	newToken.CreationTimestamp = metav1.NewTime(time.Now().Add(-time.Second * 10))
	newToken.Spec.TTLSeconds = 1
	return newToken
}
