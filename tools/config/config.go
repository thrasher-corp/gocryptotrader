package main

import (
	"flag"
	"log"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
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
	var err error
	configFile := config.GetFilePath("")
	flag.StringVar(&inFile, "infile", configFile, "The config input file to process.")
	flag.StringVar(&outFile, "outfile", configFile+".out", "The config output file.")
	flag.BoolVar(&encrypt, "encrypt", true, "Whether to encrypt or decrypt.")
	flag.StringVar(&key, "key", "", "The key to use for AES encryption.")
	flag.Parse()

	log.Println("GoCryptoTrader: config-helper tool.")

	if key == "" {
		result, errf := config.PromptForConfigKey()
		if errf != nil {
			log.Fatal("Unable to obtain encryption/decryption key.")
		}
		key = string(result)
	}

	file, err := common.ReadFile(inFile)
	if err != nil {
		log.Fatalf("Unable to read input file %s. Error: %s.", inFile, err)
	}

	if config.ConfirmECS(file) && encrypt {
		log.Println("File is already encrypted. Decrypting..")
		encrypt = false
	}

	if !config.ConfirmECS(file) && !encrypt {
		var result interface{}
		errf := config.ConfirmConfigJSON(file, result)
		if errf != nil {
			log.Fatal("File isn't in JSON format")
		}
		log.Println("File is already decrypted. Encrypting..")
		encrypt = true
	}

	var data []byte
	if encrypt {
		data, err = config.EncryptConfigFile(file, []byte(key))
		if err != nil {
			log.Fatalf("Unable to encrypt config data. Error: %s.", err)
		}
	} else {
		data, err = config.DecryptConfigFile(file, []byte(key))
		if err != nil {
			log.Fatalf("Unable to decrypt config data. Error: %s.", err)
		}
	}

	err = common.WriteFile(outFile, data)
	if err != nil {
		log.Fatalf("Unable to write output file %s. Error: %s", outFile, err)
	}
	log.Printf(
		"Successfully %s input file %s and wrote output to %s.\n",
		EncryptOrDecrypt(encrypt), inFile, outFile,
	)
}
