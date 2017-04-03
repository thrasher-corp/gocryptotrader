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
	CONFIG_ENCRYPTION_CONFIRMATION_STRING = "THORS-HAMMER"

	ErrConfigDataLessThenRequiredAESBlockSize = "The config file data is too small for the AES required block size."
)

func (c *Config) PromptForConfigEncryption() bool {
	log.Println("Would you like to encrypt your config file (y/n)?")

	input := ""
	_, err := fmt.Scanln(&input)
	if err != nil {
		return false
	}

	if !common.YesOrNo(input) {
		c.EncryptConfig = CONFIG_FILE_ENCRYPTION_DISABLED
		c.SaveConfig()
		return false
	}
	return true
}

func PromptForConfigKey() ([]byte, error) {
	var cryptoKey []byte

	for len(cryptoKey) != 32 {
		fmt.Println("Enter password (32 characters):")

		_, err := fmt.Scanln(&cryptoKey)
		if err != nil {
			return nil, err
		}

		if len(cryptoKey) > 32 || len(cryptoKey) < 32 {
			fmt.Println("Please re-enter password (32 characters):")
		}
	}
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return cryptoKey, nil
}

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

	appendedFile := []byte(CONFIG_ENCRYPTION_CONFIRMATION_STRING)
	appendedFile = append(appendedFile, ciphertext...)
	return appendedFile, nil
}

func DecryptConfigFile(configData, key []byte) ([]byte, error) {
	configData = RemoveECS(configData)
	blockDecrypt, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(configData) < aes.BlockSize {
		return nil, errors.New(ErrConfigDataLessThenRequiredAESBlockSize)
	}

	iv := configData[:aes.BlockSize]
	configData = configData[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(blockDecrypt, iv)
	stream.XORKeyStream(configData, configData)
	result := configData
	return result, nil
}

func ConfirmConfigJSON(file []byte, result interface{}) error {
	if !common.StringContains(reflect.TypeOf(result).String(), "*") {
		return errors.New("ConfirmConfigJSON Error: Parameter interface is not a pointer.")
	}
	return common.JSONDecode(file, &result)
}

func ConfirmECS(file []byte) bool {
	subslice := []byte(CONFIG_ENCRYPTION_CONFIRMATION_STRING)
	return bytes.Contains(file, subslice)
}

func RemoveECS(file []byte) []byte {
	return bytes.Trim(file, CONFIG_ENCRYPTION_CONFIRMATION_STRING)
}
