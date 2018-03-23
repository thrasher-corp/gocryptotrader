package main

import (
	"flag"
	"go/build"
	"os"

	"github.com/thrasher-/gocryptotrader/config"
	platformServices "github.com/thrasher-/gocryptotrader/platform/service"
)

func main() {
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		goPath = build.Default.GOPATH
	}

	configFile := config.GetFilePath("")

	inFile := flag.String("infile", configFile, "-infile <filepath> specifies the location of the GoCryptoTrader configuration file")
	outFile := flag.String("outfile", configFile+".out", "-outfile <filepath> specifies the output file")
	dryrun := flag.Bool("dryrun", false, "-dryrun flag does not save configuration file when GoCryptoTrader is shutdown")
	version := flag.Bool("version", false, "-version flag retrieves current GoCryptoTrader version")
	verbose := flag.Bool("V", false, "-V flag sets GoCryptoTader verbosity")
	config := flag.Bool("config", false, "-config flag starts the GoCryptoTader configuration tool")
	key := flag.String("key", "", "-key <keyphrase> adds the encryption keyphrase for the GoCryptoTrader configuration tool using AES encryption")
	encrypt := flag.Bool("encrypt", false, "-encrypt flag encrypts your configuration file")
	documentation := flag.Bool("document", false, "-document flag regenerates full documentation across the GoCryptoTrader codebase")
	createExchange := flag.String("createexchange", "", "-createexchange <exchange name> creates a new template for exchange API integration on GoCryptoTrader")
	websocketSupport := flag.Bool("ws", false, "-ws flag used in conjunction with -createexchange <exchange name> adds websocket support to template")
	restSupport := flag.Bool("rs", false, "-rs flag used in conjunction with -createexchange <exchange name> adds REST support to template")
	fixSupport := flag.Bool("fs", false, "-fs flag used in conjunction with -createexchange <exchange name> adds FIX support to template")
	portfolio := flag.Bool("portfolio", false, "-portfolio flag prints out current portfolio values associated")
	websocket := flag.Bool("websocket", false, "-websocket flag starts a websocket client")

	flag.Parse()

	if *version {
		platformServices.Version()
	}

	if *config {
		platformServices.StartConfig(*inFile, *outFile, *key, *encrypt)
	}

	if *documentation {
		platformServices.StartDocumentation(*verbose, goPath)
	}

	if *createExchange != "" {
		platformServices.StartExchangeTemplate(*createExchange, goPath, *websocketSupport, *restSupport, *fixSupport)
	}

	if *portfolio {
		platformServices.StartPortfolio(*inFile, *key)
	}

	if *websocket {
		platformServices.StartWebsocketClient()
	}

	platformServices.StartDefault(*inFile, *verbose, *dryrun)
}
