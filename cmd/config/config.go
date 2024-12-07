package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/config"
)

var commands = []string{"upgrade", "encrypt", "decrypt"}

func main() {
	fmt.Println("GoCryptoTrader: config-helper tool")

	defaultCfgFile := config.DefaultFilePath()

	var in, out, keyStr string
	var inplace bool

	fs := flag.NewFlagSet("config", flag.ExitOnError)
	fs.Usage = func() { usage(fs) }
	fs.StringVar(&in, "in", defaultCfgFile, "The config input file to process")
	fs.StringVar(&out, "out", "[in].out", "The config output file")
	fs.BoolVar(&inplace, "edit", false, "Edit; Save result to the original file")
	fs.StringVar(&keyStr, "key", "", "The key to use for AES encryption")

	cmd, args := parseCommand(os.Args[1:])
	if cmd == "" {
		usage(fs)
		os.Exit(2)
	}

	if err := fs.Parse(args); err != nil {
		fatal(err.Error())
	}

	if inplace {
		out = in
	} else if out == "[in].out" {
		out = in + ".out"
	}

	key := []byte(keyStr)
	var err error
	switch cmd {
	case "upgrade":
		err = upgradeFile(in, out, key)
	case "decrypt":
		err = encryptWrapper(in, out, key, false, decryptFile)
	case "encrypt":
		err = encryptWrapper(in, out, key, true, encryptFile)
	}

	if err != nil {
		fatal(err.Error())
	}

	fmt.Println("Success! File written to " + out)
}

func upgradeFile(in, out string, key []byte) error {
	c := &config.Config{
		EncryptionKeyProvider: func(_ bool) ([]byte, error) {
			if len(key) != 0 {
				return key, nil
			}
			return config.PromptForConfigKey(false)
		},
	}

	if err := c.ReadConfigFromFile(in, true); err != nil {
		return err
	}

	return c.SaveConfigToFile(out)
}

type encryptFunc func(string, []byte) ([]byte, error)

func encryptWrapper(in, out string, key []byte, confirmKey bool, fn encryptFunc) error {
	if len(key) == 0 {
		var err error
		if key, err = config.PromptForConfigKey(confirmKey); err != nil {
			return err
		}
	}
	outData, err := fn(in, key)
	if err != nil {
		return err
	}
	if err := file.Write(out, outData); err != nil {
		return fmt.Errorf("unable to write output file %s; Error: %w", out, err)
	}
	return nil
}

func encryptFile(in string, key []byte) ([]byte, error) {
	if config.IsFileEncrypted(in) {
		return nil, errors.New("file is already encrypted")
	}
	outData, err := config.EncryptConfigFile(readFile(in), key)
	if err != nil {
		return nil, fmt.Errorf("unable to encrypt config data. Error: %w", err)
	}
	return outData, nil
}

func decryptFile(in string, key []byte) ([]byte, error) {
	if !config.IsFileEncrypted(in) {
		return nil, errors.New("file is already decrypted")
	}
	outData, err := config.DecryptConfigFile(readFile(in), key)
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt config data. Error: %w", err)
	}
	if outData, err = jsonparser.Set(outData, []byte("-1"), "encryptConfig"); err != nil {
		return nil, fmt.Errorf("unable to decrypt config data. Error: %w", err)
	}
	return outData, nil
}

func readFile(in string) []byte {
	fileData, err := os.ReadFile(in)
	if err != nil {
		fatal("Unable to read input file " + in + "; Error: " + err.Error())
	}
	return fileData
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(2)
}

// parseCommand will return the single non-flag parameter from os.Args, and return the remaining args
// If none is provided, too many, usage() will be called and exit 1
func parseCommand(a []string) (cmd string, args []string) {
	cmds, rem := []string{}, []string{}
	for _, s := range a {
		if slices.Contains(commands, s) {
			cmds = append(cmds, s)
		} else {
			rem = append(rem, s)
		}
	}
	switch len(cmds) {
	case 0:
		fmt.Fprintln(os.Stderr, "No command provided")
	case 1: //
		return cmds[0], rem
	default:
		fmt.Fprintln(os.Stderr, "Too many commands provided: "+strings.Join(cmds, ", "))
	}
	return "", nil
}

// usage prints command usage and exits 1
func usage(fs *flag.FlagSet) {
	//nolint:dupword // deliberate duplication  of commands
	fmt.Fprintln(os.Stderr, `
Usage:
config [arguments] <command>

The commands are:
	encrypt 	encrypt infile and write to outfile
	decrypt 	decrypt infile and write to outfile
	upgrade 	upgrade the version of a decrypted config file

The arguments are:`)
	fs.PrintDefaults()
}
