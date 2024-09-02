package hashers

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

func TestBasicSha3Hash(t *testing.T) {
	secretKey, err := utils.GenerateToken()
	require.Nil(t, err)

	hasher := NewHasher()
	hash, err := hasher.CreateHash(secretKey)
	require.Nil(t, err)
	require.NotNil(t, hash)

	splitHash := strings.Split(hash, ":")
	require.Len(t, splitHash, hashSplits)
	require.Equal(t, strconv.Itoa(int(SHA3Version)), splitHash[0][1:])

	// validate the key
	require.Nil(t, hasher.VerifyHash(hash, secretKey))
	require.NotNil(t, hasher.VerifyHash(hash, "incorrect"))
}

func TestSHA3VerifyHash(t *testing.T) {
	tests := []struct {
		name      string
		hash      string
		secretKey string
		wantError bool
	}{
		{
			name:      "valid hash",
			hash:      "$1:oczLtwQYKDU:ikEs/GByhOaql/rjPGkqo+c+uYbGDT3eLrMgpu+sWEAbCGwZPn9PrgPvaM4ZiTzpQPR+beYdYDNT7t/VT11t6w",
			secretKey: "bcdfghjklmnpqrstvwxz2456789",
			wantError: false,
		},
		{
			name:      "invalid hash format",
			hash:      "$1:oczLtwQYKDU",
			secretKey: "bcdfghjklmnpqrstvwxz2456789",
			wantError: true,
		},
		{
			name:      "invalid hash version",
			hash:      "$2:oczLtwQYKDU:ikEs/GByhOaql/rjPGkqo+c+uYbGDT3eLrMgpu+sWEAbCGwZPn9PrgPvaM4ZiTzpQPR+beYdYDNT7t/VT11t6w",
			secretKey: "bcdfghjklmnpqrstvwxz2456789",
			wantError: true,
		},
		{
			name:      "invalid secret key",
			hash:      "$1:oczLtwQYKDU:ikEs/GByhOaql/rjPGkqo+c+uYbGDT3eLrMgpu+sWEAbCGwZPn9PrgPvaM4ZiTzpQPR+beYdYDNT7t/VT11t6",
			secretKey: "badkey",
			wantError: true,
		},
		{
			name:      "missing $ prefix",
			hash:      "1:oczLtwQYKDU:ikEs/GByhOaql/rjPGkqo+c+uYbGDT3eLrMgpu+sWEAbCGwZPn9PrgPvaM4ZiTzpQPR+beYdYDNT7t/VT11t6w",
			secretKey: "bcdfghjklmnpqrstvwxz2456789",
			wantError: true,
		},
		{
			name:      "non-int hash version",
			hash:      "$A:oczLtwQYKDU:ikEs/GByhOaql/rjPGkqo+c+uYbGDT3eLrMgpu+sWEAbCGwZPn9PrgPvaM4ZiTzpQPR+beYdYDNT7t/VT11t6w",
			secretKey: "bcdfghjklmnpqrstvwxz2456789",
			wantError: true,
		},
		{
			name:      "non base64 character in salt",
			hash:      "$1:@oczLtwQYKDU:ikEs/GByhOaql/rjPGkqo+c+uYbGDT3eLrMgpu+sWEAbCGwZPn9PrgPvaM4ZiTzpQPR+beYdYDNT7t/VT11t6w",
			secretKey: "bcdfghjklmnpqrstvwxz2456789",
			wantError: true,
		},
		{
			name:      "non base64 character in hash",
			hash:      "$1:oczLtwQYKDU:ikEs/GByhOaql/@rjPGkqo+c+uYbGDT3eLrMgpu+sWEAbCGwZPn9PrgPvaM4ZiTzpQPR+beYdYDNT7t/VT11t6w",
			secretKey: "bcdfghjklmnpqrstvwxz2456789",
			wantError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hasher := NewHasher()
			err := hasher.VerifyHash(test.hash, test.secretKey)
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
