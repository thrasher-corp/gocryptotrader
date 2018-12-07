package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
)

const (
	commonPath                      = "..%s..%scommon%s"
	communicationsPath              = "..%s..%scommunications%s"
	communicationsBasePath          = "..%s..%scommunications%sbase%s"
	communicationsSlackPath         = "..%s..%scommunications%sslack%s"
	communicationsSmsglobalPath     = "..%s..%scommunications%ssmsglobal%s"
	communicationsSMTPPath          = "..%s..%scommunications%ssmtpservice%s"
	communicationsTelegramPath      = "..%s..%scommunications%stelegram%s"
	configPath                      = "..%s..%sconfig%s"
	currencyPath                    = "..%s..%scurrency%s"
	currencyFXPath                  = "..%s..%scurrency%sforexprovider%s"
	currencyFXBasePath              = "..%s..%scurrency%sforexprovider%sbase%s"
	currencyFXCurrencyConverterPath = "..%s..%scurrency%sforexprovider%scurrencyconverterapi%s"
	currencyFXCurrencylayerPath     = "..%s..%scurrency%sforexprovider%scurrencylayer%s"
	currencyFXFixerPath             = "..%s..%scurrency%sforexprovider%sfixer.io%s"
	currencyFXOpenExchangeRatesPath = "..%s..%scurrency%sforexprovider%sopenexchangerates%s"
	currencyPairPath                = "..%s..%scurrency%spair%s"
	currencySymbolPath              = "..%s..%scurrency%ssymbol%s"
	currencyTranslationPath         = "..%s..%scurrency%stranslation%s"
	eventsPath                      = "..%s..%sevents%s"
	exchangesPath                   = "..%s..%sexchanges%s"
	exchangesNoncePath              = "..%s..%sexchanges%snonce%s"
	exchangesOrderbookPath          = "..%s..%sexchanges%sorderbook%s"
	exchangesStatsPath              = "..%s..%sexchanges%sstats%s"
	exchangesTickerPath             = "..%s..%sexchanges%sticker%s"
	exchangesOrdersPath             = "..%s..%sexchanges%sorders%s"
	exchangesRequestPath            = "..%s..%sexchanges%srequest%s"
	portfolioPath                   = "..%s..%sportfolio%s"
	testdataPath                    = "..%s..%stestdata%s"
	toolsPath                       = "..%s..%stools%s"
	webPath                         = "..%s..%sweb%s"
	rootPath                        = "..%s..%s"

	// exchange packages
	alphapoint    = "..%s..%sexchanges%salphapoint%s"
	anx           = "..%s..%sexchanges%sanx%s"
	binance       = "..%s..%sexchanges%sbinance%s"
	bitfinex      = "..%s..%sexchanges%sbitfinex%s"
	bitflyer      = "..%s..%sexchanges%sbitflyer%s"
	bithumb       = "..%s..%sexchanges%sbithumb%s"
	bitmex        = "..%s..%sexchanges%sbitmex%s"
	bitstamp      = "..%s..%sexchanges%sbitstamp%s"
	bittrex       = "..%s..%sexchanges%sbittrex%s"
	btcc          = "..%s..%sexchanges%sbtcc%s"
	btcmarkets    = "..%s..%sexchanges%sbtcmarkets%s"
	coinbasepro   = "..%s..%sexchanges%scoinbasepro%s"
	coinut        = "..%s..%sexchanges%scoinut%s"
	exmo          = "..%s..%sexchanges%sexmo%s"
	gateio        = "..%s..%sexchanges%sgateio%s"
	gemini        = "..%s..%sexchanges%sgemini%s"
	hitbtc        = "..%s..%sexchanges%shitbtc%s"
	huobi         = "..%s..%sexchanges%shuobi%s"
	huobihadax    = "..%s..%sexchanges%shuobihadax%s"
	itbit         = "..%s..%sexchanges%sitbit%s"
	kraken        = "..%s..%sexchanges%skraken%s"
	lakebtc       = "..%s..%sexchanges%slakebtc%s"
	liqui         = "..%s..%sexchanges%sliqui%s"
	localbitcoins = "..%s..%sexchanges%slocalbitcoins%s"
	okcoin        = "..%s..%sexchanges%sokcoin%s"
	okex          = "..%s..%sexchanges%sokex%s"
	poloniex      = "..%s..%sexchanges%spoloniex%s"
	wex           = "..%s..%sexchanges%swex%s"
	yobit         = "..%s..%sexchanges%syobit%s"
	zb            = "..%s..%sexchanges%szb%s"

	contributorsList = "https://api.github.com/repos/thrasher-/gocryptotrader/contributors"

	licenseName     = "LICENSE"
	contributorName = "CONTRIBUTORS"
)

var (
	verbose, replace     bool
	codebasePaths        map[string]string
	codebaseTemplatePath map[string]string
	codebaseReadme       map[string]readme
	tmpl                 *template.Template
	path                 string
	contributors         []contributor
)

type readme struct {
	Name         string
	Contributors []contributor
	NameURL      string
	Year         int
	CapitalName  string
}

type contributor struct {
	Login         string `json:"login"`
	URL           string `json:"html_url"`
	Contributions int    `json:"contributions"`
}

func main() {
	flag.BoolVar(&verbose, "v", false, "-v Verbose flag prints more information to the std output")
	flag.BoolVar(&replace, "r", false, "-r Replace flag generates and replaces all documentation across the code base")
	flag.Parse()

	fmt.Println(`
              GoCryptoTrader: Exchange documentation tool

    This will update and regenerate documentation for the different packages
    in GoCryptoTrader.`)

	codebasePaths = make(map[string]string)
	codebaseTemplatePath = make(map[string]string)
	codebaseReadme = make(map[string]readme)
	path = common.GetOSPathSlash()

	if err := getContributorList(); err != nil {
		log.Fatal("GoCryptoTrader: Exchange documentation tool GET error ", err)
	}

	fmt.Println("Contributor list fetched")

	if err := addTemplates(); err != nil {
		log.Fatal("GoCryptoTrader: Exchange documentation tool add template error ", err)
	}

	fmt.Println("Templates parsed")

	if err := updateReadme(); err != nil {
		log.Fatal("GoCryptoTrader: Exchange documentation tool update readme error ", err)
	}

	fmt.Println("\nTool finished")
}

// updateReadme iterates through codebase paths to check for readme files and either adds
// or replaces with new readme files.
func updateReadme() error {
	addPaths()

	for packageName := range codebasePaths {
		addReadmeData(packageName)

		if !checkReadme(packageName) {
			if verbose {
				fmt.Printf("* %s Readme file FOUND.\n", packageName)
			}
			if replace {
				if verbose {
					fmt.Println("file replacement")
				}
				if err := replaceReadme(packageName); err != nil {
					return err
				}
				continue
			}
			continue
		}
		if verbose {
			fmt.Printf("* %s Readme file NOT FOUND.\n", packageName)
		}
		if replace {
			if verbose {
				log.Println("file creation")
			}
			if err := createReadme(packageName); err != nil {
				return err
			}
			continue
		}
	}
	return nil
}

// addPaths adds paths to different potential README.md files in the codebase
func addPaths() {
	codebasePaths["common"] = fmt.Sprintf(commonPath, path, path, path)

	codebasePaths["communications comms"] = fmt.Sprintf(communicationsPath, path, path, path)
	codebasePaths["communications base"] = fmt.Sprintf(communicationsBasePath, path, path, path, path)
	codebasePaths["communications slack"] = fmt.Sprintf(communicationsSlackPath, path, path, path, path)
	codebasePaths["communications smsglobal"] = fmt.Sprintf(communicationsSmsglobalPath, path, path, path, path)
	codebasePaths["communications smtp"] = fmt.Sprintf(communicationsSMTPPath, path, path, path, path)
	codebasePaths["communications telegram"] = fmt.Sprintf(communicationsTelegramPath, path, path, path, path)

	codebasePaths["config"] = fmt.Sprintf(configPath, path, path, path)

	codebasePaths["currency"] = fmt.Sprintf(currencyPath, path, path, path)
	codebasePaths["currency forexprovider"] = fmt.Sprintf(currencyFXPath, path, path, path, path)
	codebasePaths["currency forexprovider base"] = fmt.Sprintf(currencyFXBasePath, path, path, path, path, path)
	codebasePaths["currency forexprovider currencyconverter"] = fmt.Sprintf(currencyFXCurrencyConverterPath, path, path, path, path, path)
	codebasePaths["currency forexprovider currencylayer"] = fmt.Sprintf(currencyFXCurrencylayerPath, path, path, path, path, path)
	codebasePaths["currency forexprovider fixer"] = fmt.Sprintf(currencyFXFixerPath, path, path, path, path, path)
	codebasePaths["currency forexprovider openexchangerates"] = fmt.Sprintf(currencyFXOpenExchangeRatesPath, path, path, path, path, path)
	codebasePaths["currency pair"] = fmt.Sprintf(currencyPairPath, path, path, path, path)
	codebasePaths["currency symbol"] = fmt.Sprintf(currencySymbolPath, path, path, path, path)
	codebasePaths["currency translation"] = fmt.Sprintf(currencyTranslationPath, path, path, path, path)

	codebasePaths["events"] = fmt.Sprintf(eventsPath, path, path, path)

	codebasePaths["portfolio"] = fmt.Sprintf(portfolioPath, path, path, path)
	codebasePaths["testdata"] = fmt.Sprintf(testdataPath, path, path, path)
	codebasePaths["tools"] = fmt.Sprintf(toolsPath, path, path, path)
	codebasePaths["web"] = fmt.Sprintf(webPath, path, path, path)
	codebasePaths["root"] = fmt.Sprintf(rootPath, path, path)

	codebasePaths["exchanges"] = fmt.Sprintf(exchangesPath, path, path, path)
	codebasePaths["exchanges nonce"] = fmt.Sprintf(exchangesNoncePath, path, path, path, path)
	codebasePaths["exchanges orderbook"] = fmt.Sprintf(exchangesOrderbookPath, path, path, path, path)
	codebasePaths["exchanges stats"] = fmt.Sprintf(exchangesStatsPath, path, path, path, path)
	codebasePaths["exchanges ticker"] = fmt.Sprintf(exchangesTickerPath, path, path, path, path)
	codebasePaths["exchanges orders"] = fmt.Sprintf(exchangesOrdersPath, path, path, path, path)
	codebasePaths["exchanges request"] = fmt.Sprintf(exchangesRequestPath, path, path, path, path)

	codebasePaths["exchanges alphapoint"] = fmt.Sprintf(alphapoint, path, path, path, path)
	codebasePaths["exchanges anx"] = fmt.Sprintf(anx, path, path, path, path)
	codebasePaths["exchanges binance"] = fmt.Sprintf(binance, path, path, path, path)
	codebasePaths["exchanges bitfinex"] = fmt.Sprintf(bitfinex, path, path, path, path)
	codebasePaths["exchanges bitflyer"] = fmt.Sprintf(bitflyer, path, path, path, path)
	codebasePaths["exchanges bithumb"] = fmt.Sprintf(bithumb, path, path, path, path)
	codebasePaths["exchanges bitmex"] = fmt.Sprintf(bitmex, path, path, path, path)
	codebasePaths["exchanges bitstamp"] = fmt.Sprintf(bitstamp, path, path, path, path)
	codebasePaths["exchanges bittrex"] = fmt.Sprintf(bittrex, path, path, path, path)
	codebasePaths["exchanges btcc"] = fmt.Sprintf(btcc, path, path, path, path)
	codebasePaths["exchanges btcmarkets"] = fmt.Sprintf(btcmarkets, path, path, path, path)
	codebasePaths["exchanges coinut"] = fmt.Sprintf(coinut, path, path, path, path)
	codebasePaths["exchanges exmo"] = fmt.Sprintf(exmo, path, path, path, path)
	codebasePaths["exchanges coinbasepro"] = fmt.Sprintf(coinbasepro, path, path, path, path)
	codebasePaths["exchanges gateio"] = fmt.Sprintf(gateio, path, path, path, path)
	codebasePaths["exchanges gemini"] = fmt.Sprintf(gemini, path, path, path, path)
	codebasePaths["exchanges hitbtc"] = fmt.Sprintf(hitbtc, path, path, path, path)
	codebasePaths["exchanges huobi"] = fmt.Sprintf(huobi, path, path, path, path)
	codebasePaths["exchanges huobihadax"] = fmt.Sprintf(huobihadax, path, path, path, path)
	codebasePaths["exchanges itbit"] = fmt.Sprintf(itbit, path, path, path, path)
	codebasePaths["exchanges kraken"] = fmt.Sprintf(kraken, path, path, path, path)
	codebasePaths["exchanges lakebtc"] = fmt.Sprintf(lakebtc, path, path, path, path)
	codebasePaths["exchanges liqui"] = fmt.Sprintf(liqui, path, path, path, path)
	codebasePaths["exchanges localbitcoins"] = fmt.Sprintf(localbitcoins, path, path, path, path)
	codebasePaths["exchanges okcoin"] = fmt.Sprintf(okcoin, path, path, path, path)
	codebasePaths["exchanges okex"] = fmt.Sprintf(okex, path, path, path, path)
	codebasePaths["exchanges poloniex"] = fmt.Sprintf(poloniex, path, path, path, path)
	codebasePaths["exchanges wex"] = fmt.Sprintf(wex, path, path, path, path)
	codebasePaths["exchanges yobit"] = fmt.Sprintf(yobit, path, path, path, path)
	codebasePaths["exchanges zb"] = fmt.Sprintf(zb, path, path, path, path)

	codebasePaths["CONTRIBUTORS"] = fmt.Sprintf(rootPath, path, path)
	codebasePaths["LICENSE"] = fmt.Sprintf(rootPath, path, path)
}

func addReadmeData(packageName string) {
	readmeInfo := readme{
		Name:         getName(packageName, false),
		Contributors: contributors,
		NameURL:      getslashFromName(packageName),
		Year:         time.Now().Year(),
		CapitalName:  getName(packageName, true),
	}
	codebaseReadme[packageName] = readmeInfo
}

func getName(name string, capital bool) string {
	newStrings := strings.Split(name, " ")
	if len(newStrings) > 1 {
		if capital {
			return getCapital(newStrings[1])
		}
		return newStrings[1]
	}
	if capital {
		return getCapital(name)
	}
	return name
}

func getCapital(name string) string {
	cap := strings.ToUpper(string(name[0]))
	last := name[1:]

	return cap + last
}

// getslashFromName returns a string for godoc package names
func getslashFromName(packageName string) string {
	if strings.Contains(packageName, " ") {
		s := strings.Split(packageName, " ")
		return strings.Join(s, "/")
	}
	if packageName == "testdata" || packageName == "tools" || packageName == contributorName || packageName == licenseName {
		return ""
	}
	return packageName
}

var globS = []string{
	fmt.Sprintf("common_templates%s*", common.GetOSPathSlash()),
	fmt.Sprintf("communications_templates%s*", common.GetOSPathSlash()),
	fmt.Sprintf("config_templates%s*", common.GetOSPathSlash()),
	fmt.Sprintf("currency_templates%s*", common.GetOSPathSlash()),
	fmt.Sprintf("events_templates%s*", common.GetOSPathSlash()),
	fmt.Sprintf("exchanges_templates%s*", common.GetOSPathSlash()),
	fmt.Sprintf("portfolio_templates%s*", common.GetOSPathSlash()),
	fmt.Sprintf("root_templates%s*", common.GetOSPathSlash()),
	fmt.Sprintf("sub_templates%s*", common.GetOSPathSlash()),
	fmt.Sprintf("testdata_templates%s*", common.GetOSPathSlash()),
	fmt.Sprintf("tools_templates%s*", common.GetOSPathSlash()),
	fmt.Sprintf("web_templates%s*", common.GetOSPathSlash()),
}

// addTemplates adds all the template files
func addTemplates() error {
	tmpl = template.New("")

	for _, s := range globS {
		_, err := tmpl.ParseGlob(s)
		if err != nil {
			return err
		}
	}
	return nil
}

// checkReadme checks to see if the file exists
func checkReadme(packageName string) bool {
	if packageName == licenseName || packageName == contributorName {
		_, err := os.Stat(codebasePaths[packageName] + packageName)
		return os.IsNotExist(err)
	}
	_, err := os.Stat(codebasePaths[packageName] + "README.md")
	return os.IsNotExist(err)
}

// replaceReadme replaces readme file
func replaceReadme(packageName string) error {
	if packageName == licenseName || packageName == contributorName {
		if err := deleteFile(codebasePaths[packageName] + packageName); err != nil {
			return err
		}
		return createReadme(packageName)
	}
	if err := deleteFile(codebasePaths[packageName] + "README.md"); err != nil {
		return err
	}
	return createReadme(packageName)
}

// createReadme creates new readme file and executes template
func createReadme(packageName string) error {
	if packageName == licenseName || packageName == contributorName {
		file, err := os.Create(codebasePaths[packageName] + packageName)
		defer file.Close()
		if err != nil {
			return err
		}
		if verbose {
			fmt.Println("File done")
		}
		return tmpl.ExecuteTemplate(file, packageName, codebaseReadme[packageName])
	}
	file, err := os.Create(codebasePaths[packageName] + "README.md")
	defer file.Close()
	if err != nil {
		return err
	}
	if verbose {
		fmt.Println("File done")
	}
	return tmpl.ExecuteTemplate(file, packageName, codebaseReadme[packageName])
}

func deleteFile(path string) error {
	return os.Remove(path)
}

func getContributorList() error {
	return common.SendHTTPGetRequest(contributorsList, true, false, &contributors)
}
