package config

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"

	"github.com/thrasher-/gocryptotrader/common"
)

const (
	// EncryptConfirmString has a the general confirmation string to allow us to
	// see if the file is correctly encrypted
	EncryptConfirmString = "THORS-HAMMER"
	errAESBlockSize      = "The config file data is too small for the AES required block size"
	errNotAPointer       = "Error: parameter interface is not a pointer"
)

// PromptForConfigEncryption asks for encryption key
func (c *Config) PromptForConfigEncryption() bool {
	log.Println("Would you like to encrypt your config file (y/n)?")

	input := ""
	_, err := fmt.Scanln(&input)
	if err != nil {
		return false
	}

	if !common.YesOrNo(input) {
		c.EncryptConfig = configFileEncryptionDisabled
		c.SaveConfig("")
		return false
	}
	return true
}

// PromptForConfigKey asks for configuration key
func PromptForConfigKey() ([]byte, error) {
	var cryptoKey []byte

	for len(cryptoKey) != 32 {
		log.Println("Enter password (32 characters):")

		_, err := fmt.Scanln(&cryptoKey)
		if err != nil {
			return nil, err
		}

		if len(cryptoKey) > 32 || len(cryptoKey) < 32 {
			log.Println("Please re-enter password (32 characters):")
		}
	}
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return cryptoKey, nil
}

// EncryptConfigFile encrypts configuration data that is parsed in with a key
// and returns it as a byte array with an error
func EncryptConfigFile(configData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(configData))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], configData)

	appendedFile := []byte(EncryptConfirmString)
	appendedFile = append(appendedFile, ciphertext...)
	return appendedFile, nil
}

// DecryptConfigFile decrypts configuration data with the supplied key and
// returns the un-encrypted file as a byte array with an error
func DecryptConfigFile(configData, key []byte) ([]byte, error) {
	configData = RemoveECS(configData)
	blockDecrypt, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(configData) < aes.BlockSize {
		return nil, errors.New(errAESBlockSize)
	}

	iv := configData[:aes.BlockSize]
	configData = configData[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(blockDecrypt, iv)
	stream.XORKeyStream(configData, configData)
	result := configData
	return result, nil
}

// ConfirmConfigJSON confirms JSON in file
func ConfirmConfigJSON(file []byte, result interface{}) error {
	if !common.StringContains(reflect.TypeOf(result).String(), "*") {
		return errors.New(errNotAPointer)
	}
	return common.JSONDecode(file, &result)
}

// ConfirmECS confirms that the encryption confirmation string is found
func ConfirmECS(file []byte) bool {
	subslice := []byte(EncryptConfirmString)
	return bytes.Contains(file, subslice)
}

// RemoveECS removes encryption confirmation string
func RemoveECS(file []byte) []byte {
	return bytes.Trim(file, EncryptConfirmString)
}
