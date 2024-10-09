package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/config"
)

func main() {
	var inFile, outFile, key string
	var encrypt, decrypt, upgrade bool
	defaultCfgFile := config.DefaultFilePath()
	flag.StringVar(&inFile, "infile", defaultCfgFile, "The config input file to process.")
	flag.StringVar(&outFile, "outfile", "", "The config output file.")
	flag.BoolVar(&encrypt, "encrypt", false, "Encrypt the config file")
	flag.BoolVar(&decrypt, "decrypt", false, "Decrypt the config file")
	flag.BoolVar(&upgrade, "upgrade", false, "Upgrade the config file")
	flag.StringVar(&key, "key", "", "The key to use for AES encryption.")
	flag.Parse()

	if outFile == "" {
		outFile = inFile + ".out"
	}

	fmt.Println("GoCryptoTrader: config-helper tool.")

	if !(encrypt || decrypt || upgrade) {
		fatal("Must provide one of -encrypt, -decrypt or -upgrade")
	}

	if upgrade {
		doUpgrade(inFile, outFile)
		return
	}

	if key == "" {
		result, err := config.PromptForConfigKey(false)
		if err != nil {
			fatal("Unable to obtain encryption/decryption key: " + err.Error())
		}
		key = string(result)
	}

	fileData, err := os.ReadFile(inFile)
	if err != nil {
		fatal("Unable to read input file " + inFile + "; Error: " + err.Error())
	}

	switch {
	case encrypt:
		if config.IsEncrypted(fileData) {
			fatal("File is already encrypted")
		}
		if fileData, err = config.EncryptConfigFile(fileData, []byte(key)); err != nil {
			fatal("Unable to encrypt config data. Error: " + err.Error())
		}
		fmt.Println("Encrypted config file")
	case decrypt:
		if !config.IsEncrypted(fileData) {
			fatal("File is already decrypted")
		}
		if fileData, err = config.DecryptConfigFile(fileData, []byte(key)); err != nil {
			fatal("Unable to decrypt config data. Error: " + err.Error())
		}
		fmt.Println("Decrypted config file")
	}

	if err = file.Write(outFile, fileData); err != nil {
		fatal("Unable to write output file " + outFile + "; Error: " + err.Error())
	}

	fmt.Println("Success! File written to " + outFile)
}

func doUpgrade(in, out string) {
	if config.IsFileEncrypted(in) {
		fatal("Cannot upgrade an encrypted file. Please decrypt first")
	}
	c := &config.Config{}
	err := c.ReadConfigFromFile(in, true)
	if err == nil {
		err = c.SaveConfigToFile(out)
	}
	if err != nil {
		fatal(err.Error())
	}
	fmt.Println("Success! File written to ", out)
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
