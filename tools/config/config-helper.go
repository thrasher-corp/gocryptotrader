package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
)

const (
	ENCRYPTION_CONFIRMATION_STRING = "THORS-HAMMER"
)

func EncryptOrDecrypt(encrypt bool) string {
	if encrypt {
		return "encrypted"
	}
	return "decrypted"
}

func main() {
	var inFile, outFile, key string
	var encrypt bool
	var err error
	flag.StringVar(&inFile, "infile", "config.dat", "The config input file to process.")
	flag.StringVar(&outFile, "outfile", "config.dat.out", "The config output file.")
	flag.BoolVar(&encrypt, "encrypt", true, "Wether to encrypt or decrypt.")
	flag.StringVar(&key, "key", "", "The key to use for AES encryption.")
	flag.Parse()

	log.Println("GoCryptoTrader: config-helper tool.")

	if key == "" {
		result, err := PromptForConfigKey()
		if err != nil {
			log.Fatal("Unable to obtain encryption/decryption key.")
		}
		key = string(result)
	}

	file, err := ReadFile(inFile)
	if err != nil {
		log.Fatalf("Unable to read input file %s. Error: %s.", inFile, err)
	}

	if ConfirmECS(file) && encrypt {
		log.Println("File is already encrypted. Decrypting..")
		encrypt = false
	}

	if !ConfirmECS(file) && !encrypt {
		var result interface{}
		err := ConfirmConfigJSON(file, result)
		if err != nil {
			log.Fatal("File isn't in JSON format")
		}
		log.Println("File is already decrypted. Encrypting..")
		encrypt = true
	}

	var data []byte
	if encrypt {
		data, err = EncryptConfigFile(file, []byte(key))
		if err != nil {
			log.Fatalf("Unable to encrypt config data. Error: %s.", err)
		}
	} else {
		data, err = DecryptConfigFile(file, []byte(key))
		if err != nil {
			log.Fatalf("Unable to decrypt config data. Error: %s.", err)
		}
	}

	err = WriteFile(outFile, data)
	if err != nil {
		log.Fatalf("Unable to write output file %s. Error: %s", outFile, err)
	}
	log.Printf("Successfully %s input file %s and wrote output to %s.\n", EncryptOrDecrypt(encrypt), inFile, outFile)
}

func ReadFile(path string) ([]byte, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func WriteFile(file string, data []byte) error {
	err := ioutil.WriteFile(file, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func PromptForConfigKey() ([]byte, error) {
	var cryptoKey []byte

	for len(cryptoKey) != 32 {
		log.Println("Enter password (32 characters):")

		_, err := fmt.Scanln(&cryptoKey)
		if err != nil {
			return nil, err
		}

		if len(cryptoKey) > 32 || len(cryptoKey) < 32 {
			continue
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

	appendedFile := []byte(ENCRYPTION_CONFIRMATION_STRING)
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
		return nil, errors.New("The config file data is too small for the AES required block size.")
	}

	iv := configData[:aes.BlockSize]
	configData = configData[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(blockDecrypt, iv)
	stream.XORKeyStream(configData, configData)
	result := configData
	return result, nil
}

func ConfirmConfigJSON(file []byte, result interface{}) error {
	err := json.Unmarshal(file, &result)

	if err != nil {
		return err
	}

	return nil
}

func ConfirmECS(file []byte) bool {
	subslice := []byte(ENCRYPTION_CONFIRMATION_STRING)
	return bytes.Contains(file, subslice)
}

func RemoveECS(file []byte) []byte {
	return bytes.Trim(file, ENCRYPTION_CONFIRMATION_STRING)
}
