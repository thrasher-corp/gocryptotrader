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

	"github.com/thrasher-/gocryptotrader/common"
	"golang.org/x/crypto/scrypt"
)

const (
	// EncryptConfirmString has a the general confirmation string to allow us to
	// see if the file is correctly encrypted
	EncryptConfirmString = "THORS-HAMMER"
	// SaltPrefix string
	SaltPrefix = "~GCT~SO~SALTY~"
	// SaltRandomLength is the number of random bytes to append after the prefix string
	SaltRandomLength = 12

	errAESBlockSize = "The config file data is too small for the AES required block size"
)

var (
	storedSalt []byte
	sessionDK  []byte
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
func PromptForConfigKey(initialSetup bool) ([]byte, error) {
	var cryptoKey []byte

	for {
		log.Println("Please enter in your password: ")
		pwPrompt := func(i *[]byte) error {
			_, err := fmt.Scanln(i)
			if err != nil {
				return err
			}

			return nil
		}

		var p1 []byte
		err := pwPrompt(&p1)
		if err != nil {
			return nil, err
		}

		if !initialSetup {
			cryptoKey = p1
			break
		}

		var p2 []byte
		log.Println("Please re-enter your password: ")
		err = pwPrompt(&p2)
		if err != nil {
			return nil, err
		}

		if bytes.Equal(p1, p2) {
			cryptoKey = p1
			break
		} else {
			log.Printf("Passwords did not match, please try again.")
			continue
		}
	}
	return cryptoKey, nil
}

// EncryptConfigFile encrypts configuration data that is parsed in with a key
// and returns it as a byte array with an error
func EncryptConfigFile(configData, key []byte) ([]byte, error) {
	var err error

	if len(sessionDK) == 0 {
		sessionDK, err = makeNewSessionDK(key)
		if err != nil {
			return nil, err
		}
	}

	block, err := aes.NewCipher(sessionDK)
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
	appendedFile = append(appendedFile, storedSalt...)
	appendedFile = append(appendedFile, ciphertext...)
	return appendedFile, nil
}

// DecryptConfigFile decrypts configuration data with the supplied key and
// returns the un-encrypted file as a byte array with an error
func DecryptConfigFile(configData, key []byte) ([]byte, error) {
	configData = RemoveECS(configData)
	origKey := key

	if ConfirmSalt(configData) {
		salt := make([]byte, len(SaltPrefix)+SaltRandomLength)
		salt = configData[0:len(salt)]

		dk, err := getScryptDK(key, salt)
		if err != nil {
			return nil, err
		}

		configData = configData[len(salt):]
		key = dk
	}

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

	sessionDK, err = makeNewSessionDK(origKey)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ConfirmConfigJSON confirms JSON in file
func ConfirmConfigJSON(file []byte, result interface{}) error {
	return common.JSONDecode(file, &result)
}

// ConfirmSalt checks whether the encrypted data contains a salt
func ConfirmSalt(file []byte) bool {
	return bytes.Contains(file, []byte(SaltPrefix))
}

// ConfirmECS confirms that the encryption confirmation string is found
func ConfirmECS(file []byte) bool {
	return bytes.Contains(file, []byte(EncryptConfirmString))
}

// RemoveECS removes encryption confirmation string
func RemoveECS(file []byte) []byte {
	return bytes.Trim(file, EncryptConfirmString)
}

func getScryptDK(key, salt []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, errors.New("key is empty")
	}
	return scrypt.Key(key, salt, 32768, 8, 1, 32)
}

func makeNewSessionDK(key []byte) ([]byte, error) {
	var err error
	storedSalt, err = common.GetRandomSalt([]byte(SaltPrefix), SaltRandomLength)
	if err != nil {
		return nil, err
	}

	dk, err := getScryptDK(key, storedSalt)
	if err != nil {
		return nil, err
	}

	return dk, nil
}
