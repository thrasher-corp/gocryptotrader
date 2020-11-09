package config

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
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

	errAESBlockSize = "config file data is too small for the AES required block size"
)

// promptForConfigEncryption asks for encryption confirmation
// returns true if encryption was desired, false otherwise
func promptForConfigEncryption() (bool, error) {
	log.Println("Would you like to encrypt your config file (y/n)?")

	input := ""
	_, err := fmt.Scanln(&input)
	if err != nil {
		return false, err
	}

	return common.YesOrNo(input), nil
}

// Unencrypted provides the default key provider implementation for unencrypted files
func Unencrypted() ([]byte, error) {
	return nil, errors.New("encryption key was requested, no key provided")
}

// PromptForConfigKey asks for configuration key
// if initialSetup is true, the password needs to be repeated
func PromptForConfigKey(initialSetup bool) ([]byte, error) {
	var cryptoKey []byte

	for {
		log.Println("Please enter in your password: ")
		pwPrompt := func(i *[]byte) error {
			_, err := fmt.Scanln(i)
			return err
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
		}
		log.Println("Passwords did not match, please try again.")
	}
	return cryptoKey, nil
}

// EncryptConfigFile encrypts configuration data that is parsed in with a key
// and returns it as a byte array with an error
func EncryptConfigFile(configData, key []byte) ([]byte, error) {
	sessionDK, salt, err := makeNewSessionDK(key)
	if err != nil {
		return nil, err
	}
	c := &Config{
		sessionDK:  sessionDK,
		storedSalt: salt,
	}
	return c.encryptConfigFile(configData)
}

// encryptConfigFile encrypts configuration data that is parsed in with a key
// and returns it as a byte array with an error
func (c *Config) encryptConfigFile(configData []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.sessionDK)
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
	appendedFile = append(appendedFile, c.storedSalt...)
	appendedFile = append(appendedFile, ciphertext...)
	return appendedFile, nil
}

// DecryptConfigFile decrypts configuration data with the supplied key and
// returns the un-encrypted data as a byte array with an error
func DecryptConfigFile(configData, key []byte) ([]byte, error) {
	reader := bytes.NewReader(configData)
	return (&Config{}).decryptConfigData(reader, key)
}

// decryptConfigData decrypts configuration data with the supplied key and
// returns the un-encrypted data as a byte array with an error
func (c *Config) decryptConfigData(configReader io.Reader, key []byte) ([]byte, error) {
	err := skipECS(configReader)
	if err != nil {
		return nil, err
	}
	origKey := key
	configData, err := ioutil.ReadAll(configReader)
	if err != nil {
		return nil, err
	}

	if ConfirmSalt(configData) {
		salt := make([]byte, len(SaltPrefix)+SaltRandomLength)
		salt = configData[0:len(salt)]

		key, err = getScryptDK(key, salt)
		if err != nil {
			return nil, err
		}

		configData = configData[len(salt):]
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

	sessionDK, storedSalt, err := makeNewSessionDK(origKey)
	if err != nil {
		return nil, err
	}
	c.sessionDK, c.storedSalt = sessionDK, storedSalt

	return result, nil
}

// ConfirmSalt checks whether the encrypted data contains a salt
func ConfirmSalt(file []byte) bool {
	return bytes.Contains(file, []byte(SaltPrefix))
}

// ConfirmECS confirms that the encryption confirmation string is found
func ConfirmECS(file []byte) bool {
	return bytes.Contains(file, []byte(EncryptConfirmString))
}

// skipECS skips encryption confirmation string
// or errors, if the prefix wasn't found
func skipECS(file io.Reader) error {
	buf := make([]byte, len(EncryptConfirmString))
	_, err := io.ReadFull(file, buf)
	if err != nil {
		return err
	}
	if string(buf) != EncryptConfirmString {
		return errors.New("data does not start with ECS")
	}
	return nil
}

func getScryptDK(key, salt []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, errors.New("key is empty")
	}
	return scrypt.Key(key, salt, 32768, 8, 1, 32)
}

func makeNewSessionDK(key []byte) (dk, storedSalt []byte, err error) {
	storedSalt, err = crypto.GetRandomSalt([]byte(SaltPrefix), SaltRandomLength)
	if err != nil {
		return nil, nil, err
	}

	dk, err = getScryptDK(key, storedSalt)
	if err != nil {
		return nil, nil, err
	}

	return dk, storedSalt, nil
}
