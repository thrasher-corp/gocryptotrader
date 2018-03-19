package main

import (
	"flag"

	platformServices "github.com/thrasher-/gocryptotrader/platform/service"
)

func main() {
	var configFilePath string

	//Handle flags
	flag.StringVar(&configFilePath, "config", "", "-config <filepath> specifies the location of the GoCryptoTrader configuration file")
	dryrun := flag.Bool("dryrun", false, "-dryrun flag does not save configuration file when GoCryptoTrader is shutdown")
	version := flag.Bool("version", false, "-version flag retrieves current GoCryptoTrader version")
	verbose := flag.Bool("V", false, "-V sets GoCryptoTader verbosity")
	flag.Parse()

	if *version {
		platformServices.Version()
	}

	platformServices.StartDefault(configFilePath, *verbose, *dryrun)
}
