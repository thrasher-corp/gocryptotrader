package config

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"golang.org/x/crypto/scrypt"
	"golang.org/x/term"
)

const (
	saltRandomLength = 12
)

// Public errors
var (
	ErrSettingEncryptConfig = errors.New("error setting EncryptConfig during encrypt config file")
)

var (
	errAESBlockSize = errors.New("config file data is too small for the AES required block size")
	errNoPrefix     = errors.New("data does not start with Encryption Prefix")
	errKeyIsEmpty   = errors.New("key is empty")
	errUserInput    = errors.New("error getting user input")

	// encryptionPrefix is a prefix to tell us the file is encrypted
	encryptionPrefix = []byte("THORS-HAMMER")
	saltPrefix       = []byte("~GCT~SO~SALTY~")
)

// promptForConfigEncryption asks for encryption confirmation
// returns true if encryption was desired, false otherwise
func promptForConfigEncryption() (bool, error) {
	fmt.Println("Would you like to encrypt your config file (y/n)?")

	input := ""
	if _, err := fmt.Scanln(&input); err != nil {
		return false, err
	}

	return common.YesOrNo(input), nil
}

// PromptForConfigKey asks for configuration key
func PromptForConfigKey(confirmKey bool) ([]byte, error) {
	for range 3 {
		key, err := getSensitiveInput("Please enter encryption key: ")
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errUserInput, err)
		}

		if len(key) == 0 {
			continue
		}

		if !confirmKey {
			return key, nil
		}

		conf, err := getSensitiveInput("Please re-enter key: ")
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errUserInput, err)
		}

		if bytes.Equal(key, conf) {
			return key, nil
		}
		fmt.Println("Keys did not match, please try again.")
	}
	return nil, fmt.Errorf("%w: %w", errUserInput, io.EOF)
}

// getSensitiveInput reads input from stdin, with echo off if stdin is a terminal
func getSensitiveInput(prompt string) (resp []byte, err error) {
	fmt.Print(prompt)
	defer fmt.Println()
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return term.ReadPassword(int(os.Stdin.Fd()))
	}
	// Can't use bufio.* because it consumes the whole input in one go, even with s.Buffer(1)
	for buf := make([]byte, 1); err == nil && buf[0] != '\n'; {
		if _, err = os.Stdin.Read(buf); err == nil {
			resp = append(resp, buf[0])
		}
	}
	return bytes.TrimRight(resp, "\r\n"), err
}

// EncryptConfigData encrypts json config data with a key
func EncryptConfigData(configData, key []byte) ([]byte, error) {
	sessionDK, salt, err := makeNewSessionDK(key)
	if err != nil {
		return nil, err
	}
	c := &Config{
		sessionDK:  sessionDK,
		storedSalt: salt,
	}
	return c.encryptConfigData(configData)
}

// encryptConfigData encrypts json config data with a key
// The EncryptConfig field is set to config enabled (1)
func (c *Config) encryptConfigData(configData []byte) ([]byte, error) {
	configData, err := jsonparser.Set(configData, []byte("1"), "encryptConfig")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSettingEncryptConfig, err)
	}

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

	appendedFile := append(bytes.Clone(encryptionPrefix), c.storedSalt...)
	appendedFile = append(appendedFile, ciphertext...)
	return appendedFile, nil
}

// DecryptConfigData decrypts config data with a key
func DecryptConfigData(d, key []byte) ([]byte, error) {
	return (&Config{}).decryptConfigData(d, key)
}

// decryptConfigData decrypts config data with a key
func (c *Config) decryptConfigData(d, key []byte) ([]byte, error) {
	if !bytes.HasPrefix(d, encryptionPrefix) {
		return d, errNoPrefix
	}

	d = bytes.TrimPrefix(d, encryptionPrefix)

	sessionDK, storedSalt, err := makeNewSessionDK(key)
	if err != nil {
		return nil, err
	}

	if bytes.HasPrefix(d, saltPrefix) {
		salt := make([]byte, len(saltPrefix)+saltRandomLength)
		salt = d[0:len(salt)]

		key, err = getScryptDK(key, salt)
		if err != nil {
			return nil, err
		}

		d = d[len(salt):]
	}

	blockDecrypt, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(d) < aes.BlockSize {
		return nil, errAESBlockSize
	}

	iv, d := d[:aes.BlockSize], d[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(blockDecrypt, iv)
	stream.XORKeyStream(d, d)

	c.sessionDK, c.storedSalt = sessionDK, storedSalt

	return d, nil
}

// IsEncrypted returns if the data sequence is encrypted
func IsEncrypted(data []byte) bool {
	return bytes.HasPrefix(data, encryptionPrefix)
}

// IsFileEncrypted returns if the file is encrypted
// Returns false on error opening or reading
func IsFileEncrypted(f string) bool {
	r, err := os.Open(f)
	if err != nil {
		return false
	}
	defer r.Close()
	prefix := make([]byte, len(encryptionPrefix))
	if _, err = io.ReadFull(r, prefix); err != nil {
		return false
	}
	return bytes.Equal(prefix, encryptionPrefix)
}

func getScryptDK(key, salt []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, errKeyIsEmpty
	}
	return scrypt.Key(key, salt, 32768, 8, 1, 32)
}

func makeNewSessionDK(key []byte) (dk, storedSalt []byte, err error) {
	storedSalt, err = crypto.GetRandomSalt(saltPrefix, saltRandomLength)
	if err != nil {
		return nil, nil, err
	}

	dk, err = getScryptDK(key, storedSalt)
	if err != nil {
		return nil, nil, err
	}

	return dk, storedSalt, nil
}
