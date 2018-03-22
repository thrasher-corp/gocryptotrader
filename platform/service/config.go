package service

import (
	"log"
	"os"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

// StartConfig reads from inFile, decrypts file if encrypted and writes to
// a specified location (outFile)
// encrypt - denotes outFile encryption status
func StartConfig(inFile, outFile, key string, encrypt bool) {
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
		encryptOrDecrypt(encrypt), inFile, outFile,
	)
	os.Exit(0)
}

func encryptOrDecrypt(encrypt bool) string {
	if encrypt {
		return "encrypted"
	}
	return "decrypted"
}
