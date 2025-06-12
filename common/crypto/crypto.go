package crypto

import (
	"crypto/hmac"
	"crypto/md5" //nolint:gosec // Used for exchanges
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // Used for exchanges
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"hash"
	"io"
)

// Const declarations for common.go operations
const (
	HashSHA1 = iota
	HashSHA256
	HashSHA512
	HashSHA512_384
	HashMD5
)

// GetRandomSalt returns a random salt
func GetRandomSalt(input []byte, saltLen int) ([]byte, error) {
	if saltLen <= 0 {
		return nil, errors.New("salt length is too small")
	}
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	var result []byte
	if input != nil {
		result = input
	}
	result = append(result, salt...)
	return result, nil
}

// GetHMAC returns a keyed-hash message authentication code using the desired
// hashtype
func GetHMAC(hashType int, input, key []byte) ([]byte, error) {
	var hasher func() hash.Hash

	switch hashType {
	case HashSHA1:
		hasher = sha1.New
	case HashSHA256:
		hasher = sha256.New
	case HashSHA512:
		hasher = sha512.New
	case HashSHA512_384:
		hasher = sha512.New384
	case HashMD5:
		hasher = md5.New
	}

	h := hmac.New(hasher, key)
	_, err := h.Write(input)
	return h.Sum(nil), err
}
