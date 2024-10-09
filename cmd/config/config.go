package main

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/config"
)

var commands = []string{"upgrade", "encrypt", "decrypt"}

func main() {
	fmt.Println("GoCryptoTrader: config-helper tool")

	defaultCfgFile := config.DefaultFilePath()

	var in, out, key string
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	fs.Usage = func() { usage(fs) }
	fs.StringVar(&in, "infile", defaultCfgFile, "The config input file to process")
	fs.StringVar(&out, "outfile", "[infile].out", "The config output file")
	fs.StringVar(&key, "key", "", "The key to use for AES encryption")
	_ = fs.Parse(os.Args[1:])

	if out == "[infile].out" {
		out = in + ".out"
	}

	switch parseCommand(fs) {
	case "upgrade":
		upgradeFile(in, out)
	case "decrypt":
		encryptWrapper(in, out, key, decryptFile)
	case "encrypt":
		encryptWrapper(in, out, key, encryptFile)
	}

	fmt.Println("Success! File written to " + out)
}

func upgradeFile(in, out string) {
	if config.IsFileEncrypted(in) {
		fatal("Cannot upgrade an encrypted file. Please decrypt first")
	}
	c := &config.Config{}
	if err := c.ReadConfigFromFile(in, true); err != nil {
		fatal(err.Error())
	}
	if err := c.SaveConfigToFile(out); err != nil {
		fatal(err.Error())
	}
}

func encryptWrapper(in, out, key string, fn func(in string, key []byte) []byte) {
	if key == "" {
		key = getKey()
	}
	outData := fn(in, []byte(key))
	if err := file.Write(out, outData); err != nil {
		fatal("Unable to write output file " + out + "; Error: " + err.Error())
	}
}

func encryptFile(in string, key []byte) []byte {
	if config.IsFileEncrypted(in) {
		fatal("File is already encrypted")
	}
	outData, err := config.EncryptConfigFile(readFile(in), key)
	if err != nil {
		fatal("Unable to encrypt config data. Error: " + err.Error())
	}
	return outData
}

func decryptFile(in string, key []byte) []byte {
	if !config.IsFileEncrypted(in) {
		fatal("File is already decrypted")
	}
	outData, err := config.DecryptConfigFile(readFile(in), key)
	if err != nil {
		fatal("Unable to decrypt config data. Error: " + err.Error())
	}
	return outData
}

func readFile(in string) []byte {
	fileData, err := os.ReadFile(in)
	if err != nil {
		fatal("Unable to read input file " + in + "; Error: " + err.Error())
	}
	return fileData
}

func getKey() string {
	result, err := config.PromptForConfigKey(false)
	if err != nil {
		fatal("Unable to obtain encryption/decryption key: " + err.Error())
	}
	return string(result)
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(2)
}

// parseCommand will return the single non-flag parameter
// If none is provided, too many, or unrecognised, usage() will be called and exit 1
func parseCommand(fs *flag.FlagSet) string {
	switch fs.NArg() {
	case 0:
		fmt.Fprintln(os.Stderr, "No command provided")
	case 1:
		command := fs.Arg(0)
		if slices.Contains(commands, command) {
			return command
		}
		fmt.Fprintln(os.Stderr, "Unknown command provided: "+command)
	default:
		fmt.Fprintln(os.Stderr, "Too many commands provided: "+strings.Join(fs.Args(), " "))
	}
	usage(fs)
	os.Exit(2)
	return ""
}

// usage prints command usage and exits 1
func usage(fs *flag.FlagSet) {
	//nolint:dupword // deliberate duplication  of commands
	fmt.Fprintln(os.Stderr, `
Usage:
config <command> [arguments]

The commands are:
	encrypt 	encrypt infile and write to outfile
	decrypt 	decrypt infile and write to outfile
	upgrade 	upgrade the version of a decrypted config file

The arguments are:`)
	fs.PrintDefaults()
}
