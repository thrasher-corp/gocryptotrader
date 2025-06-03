package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
)

var commands = []string{"upgrade", "downgrade", "encrypt", "decrypt"}

func main() {
	fmt.Println("GoCryptoTrader: config-helper tool")

	defaultCfgFile := config.DefaultFilePath()

	var in, out, keyStr string
	var inplace bool
	var version uint

	fs := flag.NewFlagSet("config", flag.ExitOnError)
	fs.Usage = func() { usage(fs) }
	fs.StringVar(&in, "in", defaultCfgFile, "The config input file to process")
	fs.StringVar(&out, "out", "[in].out", "The config output file")
	fs.BoolVar(&inplace, "edit", false, "Edit; Save result to the original file")
	fs.StringVar(&keyStr, "key", "", "The key to use for AES encryption")
	fs.UintVar(&version, "version", 0, "The version to downgrade to")

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

	var err error
	key := []byte(keyStr)
	data := readFile(in)
	isEncrypted := config.IsEncrypted(data)

	if cmd == "encrypt" && isEncrypted {
		fatal("Error: File is already encrypted")
	}
	if cmd == "decrypt" && !isEncrypted {
		fatal("Error: File is already decrypted")
	}

	if len(key) == 0 && (isEncrypted || cmd == "encrypt") {
		if key, err = config.PromptForConfigKey(cmd == "encrypt"); err != nil {
			fatal(err.Error())
		}
	}

	if isEncrypted {
		if data, err = config.DecryptConfigData(data, key); err != nil {
			fatal(err.Error())
		}
	}

	switch cmd {
	case "decrypt":
		if data, err = jsonparser.Set(data, []byte("-1"), "encryptConfig"); err != nil {
			fatal("Unable to decrypt config data; Error: " + err.Error())
		}
	case "downgrade", "upgrade":
		if version == 0 {
			if cmd == "downgrade" {
				fmt.Fprintln(os.Stderr, "Error: downgrade requires a version")
				usage(fs)
				os.Exit(3)
			}
			version = versions.UseLatestVersion
		} else if version >= math.MaxUint16 {
			fmt.Fprintln(os.Stderr, "Error: version must be less than 65535")
			usage(fs)
			os.Exit(3)
		}
		if data, err = versions.Manager.Deploy(context.Background(), data, uint16(version)); err != nil {
			fatal("Unable to " + cmd + " config; Error: " + err.Error())
		}
		if !isEncrypted {
			break
		}
		fallthrough
	case "encrypt":
		if data, err = config.EncryptConfigData(data, key); err != nil {
			fatal("Unable to encrypt config data; Error: " + err.Error())
		}
	}

	if err := file.Write(out, data); err != nil {
		fatal("Unable to write output file `" + out + "`; Error: " + err.Error())
	}

	fmt.Println("Success! File written to " + out)
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
	case 1:
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
	downgrade 	downgrade the version of a decrypted config file to a specific version

The arguments are:`)
	fs.PrintDefaults()
}
