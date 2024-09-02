package hashers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/sha3"
)

type HashVersion int

const (
	SHA3Version HashVersion = iota + 1
)

const (
	sha3HashFormat = "$%d:%s:%s" // $version:salt:hash -> $1:abc:def
	hashSplits     = 3
	// saltLength is the length of the salt used in the hash
	saltLength = 8
)

type Sha3Hasher struct{}

func NewHasher() Sha3Hasher {
	return Sha3Hasher{}
}

// CreateHash hashes secretKey using a random salt and SHA3/SHA512.
func (s Sha3Hasher) CreateHash(secretKey string) (string, error) {
	secretKeyBytes := []byte(secretKey)
	// to save on allocations, we will first read the salt into hashInput, and then append the secretKey
	// rand will only read to len(hashInput), so we set the length to saltLength but pre-allocate the
	// capacity for the secret key that will be appended later
	hashInput := make([]byte, saltLength, saltLength+len(secretKeyBytes))
	_, err := rand.Read(hashInput)
	if err != nil {
		return "", fmt.Errorf("unable to read random values for salt: %w", err)
	}

	hashInput = append(hashInput, secretKeyBytes...)
	hash := sha3.Sum512(hashInput)
	encSalt := base64.RawStdEncoding.EncodeToString(hashInput[:saltLength])
	encKey := base64.RawStdEncoding.EncodeToString(hash[:])
	return fmt.Sprintf(sha3HashFormat, SHA3Version, encSalt, encKey), nil
}

// VerifyHash compares a key with the hash, and will produce an error if the hash does not match or if the hash is not
// a valid SHA3 hash.
func (s Sha3Hasher) VerifyHash(hash, secretKey string) error {
	if !strings.HasPrefix(hash, "$") {
		return fmt.Errorf("hash format invalid")
	}
	splitHash := strings.Split(strings.TrimPrefix(hash, "$"), ":")
	if len(splitHash) != hashSplits {
		return fmt.Errorf("hash format invalid")
	}

	version, err := strconv.Atoi(splitHash[0])
	if err != nil {
		return err
	}
	if HashVersion(version) != SHA3Version {
		return fmt.Errorf("hash version %d does not match package version %d", version, SHA3Version)
	}

	salt, enc := splitHash[1], splitHash[2]
	// base64 decode stored salt and key
	decodedKey, err := base64.RawStdEncoding.DecodeString(enc)
	if err != nil {
		return err
	}

	if len(decodedKey) < 1 {
		return fmt.Errorf("secretKey hash does not match") // Don't allow accidental empty string to succeed
	}
	decodedSalt, err := base64.RawStdEncoding.DecodeString(salt)
	if err != nil {
		return err
	}

	// Compare the keys
	hashedSecretKey := sha3.Sum512([]byte(fmt.Sprintf("%s%s", string(decodedSalt), secretKey)))
	if subtle.ConstantTimeCompare(decodedKey, hashedSecretKey[:]) == 0 {
		return fmt.Errorf("secretKey hash does not match")
	}
	return nil
}
