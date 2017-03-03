package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

const (
	ENCRYPTION_CONFIRMATION_STRING = "THORS-HAMMER"
)

type Encryption struct {
	CryptoKey     []byte
	EncryptedFile []byte
	Nonce         []byte
	encrypPerm    bool
	decryptPerm   bool
}

func (e *Encryption) SetUp() {
	for len(e.CryptoKey) != 32 {
		fmt.Println("Please enter a unique 32Char key to set up encryption: \n")

		_, err := fmt.Scanln(&e.CryptoKey)
		if err != nil {
			panic(err)
		}

		if len(e.CryptoKey) > 32 || len(e.CryptoKey) < 32 {
			fmt.Println("Please Re-enter a 32char key..\n")
		}
	}

	e.Nonce = make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, e.Nonce); err != nil {
		panic(err)
	}

	e.encrypPerm = false
	e.decryptPerm = false
}

func (e *Encryption) Encrypt() []byte {
	block, err := aes.NewCipher(e.CryptoKey)
	if err != nil {
		panic(err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}

	configfile := e.ReadFile(CONFIG_FILE)

	if e.ConfirmJSON(configfile) != true {
		log.Println("File cannot be encrypted.\n")
		if e.ConfirmECS(configfile) != true {
			log.Println("File Corrupted.")
			panic(err)
		}
		log.Println("File already encrypted.")
		return configfile
	}
	e.EncryptedFile = aesgcm.Seal(nil, e.Nonce, configfile, nil)

	appendedFile := []byte(ENCRYPTION_CONFIRMATION_STRING)
	appendedFile = append(appendedFile, e.EncryptedFile...)
	return appendedFile
}

func (e *Encryption) Decrypt() []byte {
	blockDecrypt, err := aes.NewCipher(e.CryptoKey)
	if err != nil {
		panic(err)
	}

	aesgcmDecrypt, err := cipher.NewGCM(blockDecrypt)
	if err != nil {
		panic(err)
	}

	configfile := e.ReadFile(CONFIG_FILE)
	if e.ConfirmECS(configfile) != true {
		log.Println("File cannot be decrypted..\n")
		if e.ConfirmJSON(configfile) != true {
			log.Println("File corrupted.")
			panic(err)
		}
		log.Println("File already decrypted.")
		return configfile
	}

	unencryptedFile, err := aesgcmDecrypt.Open(nil, e.Nonce, e.RemoveECS(configfile), nil)
	if err != nil {
		log.Println("File Corrupted")
		panic(err)
	}
	return unencryptedFile
}

func (e *Encryption) ReadFile(filename string) []byte {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return file
}

func (e *Encryption) SaveFile(file []byte) {
	mode := int(0777)
	perm := os.FileMode(mode)
	err := ioutil.WriteFile(CONFIG_FILE, file, perm)
	if err != nil {
		panic(err)
	}
}

func (e *Encryption) ConfirmJSON(file []byte) bool {
	err := json.Unmarshal(file, nil) //Needs Revision
	if err != nil {
		return true //Fix after revision
	}
	return true
}

func (e *Encryption) ConfirmECS(file []byte) bool {
	subslice := []byte(ENCRYPTION_CONFIRMATION_STRING)
	return bytes.Contains(file, subslice)
}

func (e *Encryption) RemoveECS(file []byte) []byte {
	return bytes.Trim(file, ENCRYPTION_CONFIRMATION_STRING)
}
