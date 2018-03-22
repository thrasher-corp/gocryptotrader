package main

import (
	"flag"

	"github.com/thrasher-/gocryptotrader/config"
	platformServices "github.com/thrasher-/gocryptotrader/platform/service"
)

func main() {
	configFile := config.GetFilePath("")

	inFile := flag.String("infile", configFile, "-infile <filepath> specifies the location of the GoCryptoTrader configuration file")
	outFile := flag.String("outfile", configFile+".out", "-outfile <filepath> specifies the output file")
	dryrun := flag.Bool("dryrun", false, "-dryrun flag does not save configuration file when GoCryptoTrader is shutdown")
	version := flag.Bool("version", false, "-version flag retrieves current GoCryptoTrader version")
	verbose := flag.Bool("V", false, "-V flag sets GoCryptoTader verbosity")
	config := flag.Bool("config", false, "-config flag starts the GoCryptoTader configuration tool")
	key := flag.String("key", "", "-key <keyphrase> adds the encryption keyphrase for the GoCryptoTrader configuration tool using AES encryption")
	encrypt := flag.Bool("encrypt", false, "-encrypt flag encrypts your configuration file")

	flag.Parse()

	if *version {
		platformServices.Version()
	}

	if *config {
		platformServices.StartConfig(*inFile, *outFile, *key, *encrypt)
	}

	platformServices.StartDefault(*inFile, *verbose, *dryrun)
}
