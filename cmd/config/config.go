package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/config"
)

// EncryptOrDecrypt returns a string from a boolean
func EncryptOrDecrypt(encrypt bool) string {
	if encrypt {
		return "encrypted"
	}
	return "decrypted"
}

func main() {
	var inFile, outFile, key string
	var encrypt bool
	defaultCfgFile := config.DefaultFilePath()
	flag.StringVar(&inFile, "infile", defaultCfgFile, "The config input file to process.")
	flag.StringVar(&outFile, "outfile", defaultCfgFile+".out", "The config output file.")
	flag.BoolVar(&encrypt, "encrypt", true, "Whether to encrypt or decrypt.")
	flag.StringVar(&key, "key", "", "The key to use for AES encryption.")
	flag.Parse()

	log.Println("GoCryptoTrader: config-helper tool.")

	if key == "" {
		result, err := config.PromptForConfigKey(false)
		if err != nil {
			log.Fatalf("Unable to obtain encryption/decryption key: %s", err)
		}
		key = string(result)
	}

	fileData, err := os.ReadFile(inFile)
	if err != nil {
		log.Fatalf("Unable to read input file %s. Error: %s.", inFile, err)
	}

	switch {
	case encrypt && config.IsEncrypted(fileData):
		log.Println("File is already encrypted. Decrypting..")
		encrypt = false
	case !encrypt && !config.IsEncrypted(fileData):
		var result interface{}
		if err = json.Unmarshal(fileData, &result); err != nil {
			log.Fatal(err)
		}
		log.Println("File is already decrypted. Encrypting..")
		encrypt = true
	}

	var data []byte
	if encrypt {
		data, err = config.EncryptConfigFile(fileData, []byte(key))
		if err != nil {
			log.Fatalf("Unable to encrypt config data. Error: %s.", err)
		}
	} else {
		data, err = config.DecryptConfigFile(fileData, []byte(key))
		if err != nil {
			log.Fatalf("Unable to decrypt config data. Error: %s.", err)
		}
	}

	err = file.Write(outFile, data)
	if err != nil {
		log.Fatalf("Unable to write output file %s. Error: %s", outFile, err)
	}
	log.Printf(
		"Successfully %s input file %s and wrote output to %s.\n",
		EncryptOrDecrypt(encrypt), inFile, outFile,
	)
}
