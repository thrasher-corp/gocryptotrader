package crypto

import (
	"crypto/hmac"
	"crypto/md5" //nolint:gosec // Used for exchanges
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // Used for exchanges
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
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

// HexEncodeToString takes in a hexadecimal byte array and returns a string
func HexEncodeToString(input []byte) string {
	return hex.EncodeToString(input)
}

// Base64Decode takes in a Base64 string and returns a byte array and an error
func Base64Decode(input string) ([]byte, error) {
	result, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Base64Encode takes in a byte array then returns an encoded base64 string
func Base64Encode(input []byte) string {
	return base64.StdEncoding.EncodeToString(input)
}

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

// GetMD5 returns a MD5 hash of a byte array
func GetMD5(input []byte) ([]byte, error) {
	m := md5.New() //nolint:gosec // hash function used by some exchanges
	_, err := m.Write(input)
	return m.Sum(nil), err
}

// GetSHA512 returns a SHA512 hash of a byte array
func GetSHA512(input []byte) ([]byte, error) {
	sha := sha512.New()
	_, err := sha.Write(input)
	return sha.Sum(nil), err
}

// GetSHA256 returns a SHA256 hash of a byte array
func GetSHA256(input []byte) ([]byte, error) {
	sha := sha256.New()
	_, err := sha.Write(input)
	return sha.Sum(nil), err
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

// Sha1ToHex takes a string, sha1 hashes it and return a hex string of the
// result
func Sha1ToHex(data string) (string, error) {
	h := sha1.New() //nolint:gosec // hash function used by some exchanges
	_, err := h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil)), err
}
