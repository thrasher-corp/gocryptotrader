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
	commonPath              = "..%s..%scommon%s"
	configPath              = "..%s..%sconfig%s"
	currencyPath            = "..%s..%scurrency%s"
	currencyPairPath        = "..%s..%scurrency%spair%s"
	currencySymbolPath      = "..%s..%scurrency%ssymbol%s"
	currencyTranslationPath = "..%s..%scurrency%stranslation%s"
	eventsPath              = "..%s..%sevents%s"
	exchangesPath           = "..%s..%sexchanges%s"
	exchangesNoncePath      = "..%s..%sexchanges%snonce%s"
	exchangesOrderbookPath  = "..%s..%sexchanges%sorderbook%s"
	exchangesStatsPath      = "..%s..%sexchanges%sstats%s"
	exchangesTickerPath     = "..%s..%sexchanges%sticker%s"
	portfolioPath           = "..%s..%sportfolio%s"
	smsglobalPath           = "..%s..%ssmsglobal%s"
	testdataPath            = "..%s..%stestdata%s"
	toolsPath               = "..%s..%stools%s"
	webPath                 = "..%s..%sweb%s"
	rootPath                = "..%s..%s"

	// exchange packages
	alphapoint    = "..%s..%sexchanges%salphapoint%s"
	anx           = "..%s..%sexchanges%sanx%s"
	binance       = "..%s..%sexchanges%sbinance%s"
	bitfinex      = "..%s..%sexchanges%sbitfinex%s"
	bitflyer      = "..%s..%sexchanges%sbitflyer%s"
	bithumb       = "..%s..%sexchanges%sbithumb%s"
	bitstamp      = "..%s..%sexchanges%sbitstamp%s"
	bittrex       = "..%s..%sexchanges%sbittrex%s"
	btcc          = "..%s..%sexchanges%sbtcc%s"
	btcmarkets    = "..%s..%sexchanges%sbtcmarkets%s"
	coinut        = "..%s..%sexchanges%scoinut%s"
	exmo          = "..%s..%sexchanges%sexmo%s"
	gdax          = "..%s..%sexchanges%sgdax%s"
	gemini        = "..%s..%sexchanges%sgemini%s"
	hitbtc        = "..%s..%sexchanges%shitbtc%s"
	huobi         = "..%s..%sexchanges%shuobi%s"
	itbit         = "..%s..%sexchanges%sitbit%s"
	kraken        = "..%s..%sexchanges%skraken%s"
	lakebtc       = "..%s..%sexchanges%slakebtc%s"
	localbitcoins = "..%s..%sexchanges%slocalbitcoins%s"
	okcoin        = "..%s..%sexchanges%sokcoin%s"
	okex          = "..%s..%sexchanges%sokex%s"
	poloniex      = "..%s..%sexchanges%spoloniex%s"
	wex           = "..%s..%sexchanges%swex%s"
	yobit         = "..%s..%sexchanges%syobit%s"
	liqui         = "..%s..%sexchanges%sliqui%s"

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
	// exchanges []string{"alphapoint", "anx", "binance", "bitfinex", "bitflyer",
	// "bithumb"}
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
    in GoCryptoTrader.
`)

	codebasePaths = make(map[string]string)
	codebaseTemplatePath = make(map[string]string)
	codebaseReadme = make(map[string]readme)
	path = common.GetOSPathSlash()

	if err := getContributorList(); err != nil {
		log.Fatal("GoCryptoTrader: Exchange documentation tool GET error ", err)
	}

	if err := addTemplates(); err != nil {
		log.Fatal("GoCryptoTrader: Exchange documentation tool add template error ", err)
	}

	if err := updateReadme(); err != nil {
		log.Fatal("GoCryptoTrader: Exchange documentation tool update readme error ", err)
	}

	fmt.Println("\nTool finished")
}

// Iterates through codebase paths to check for readme files and either adds
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

// Adds paths to different potential README.md files in the codebase
func addPaths() {
	codebasePaths["common"] = fmt.Sprintf(commonPath, path, path, path)
	codebasePaths["config"] = fmt.Sprintf(configPath, path, path, path)
	codebasePaths["currency"] = fmt.Sprintf(currencyPath, path, path, path)
	codebasePaths["currency pair"] = fmt.Sprintf(currencyPairPath, path, path, path, path)
	codebasePaths["currency symbol"] = fmt.Sprintf(currencySymbolPath, path, path, path, path)
	codebasePaths["currency translation"] = fmt.Sprintf(currencyTranslationPath, path, path, path, path)
	codebasePaths["events"] = fmt.Sprintf(eventsPath, path, path, path)
	codebasePaths["exchanges"] = fmt.Sprintf(exchangesPath, path, path, path)
	codebasePaths["exchanges nonce"] = fmt.Sprintf(exchangesNoncePath, path, path, path, path)
	codebasePaths["exchanges orderbook"] = fmt.Sprintf(exchangesOrderbookPath, path, path, path, path)
	codebasePaths["exchanges stats"] = fmt.Sprintf(exchangesStatsPath, path, path, path, path)
	codebasePaths["exchanges ticker"] = fmt.Sprintf(exchangesTickerPath, path, path, path, path)
	codebasePaths["portfolio"] = fmt.Sprintf(portfolioPath, path, path, path)
	codebasePaths["smsglobal"] = fmt.Sprintf(smsglobalPath, path, path, path)
	codebasePaths["testdata"] = fmt.Sprintf(testdataPath, path, path, path)
	codebasePaths["tools"] = fmt.Sprintf(toolsPath, path, path, path)
	codebasePaths["web"] = fmt.Sprintf(webPath, path, path, path)
	codebasePaths["root"] = fmt.Sprintf(rootPath, path, path)

	codebasePaths["exchanges alphapoint"] = fmt.Sprintf(alphapoint, path, path, path, path)
	codebasePaths["exchanges anx"] = fmt.Sprintf(anx, path, path, path, path)
	codebasePaths["exchanges binance"] = fmt.Sprintf(binance, path, path, path, path)
	codebasePaths["exchanges bitfinex"] = fmt.Sprintf(bitfinex, path, path, path, path)
	codebasePaths["exchanges bitflyer"] = fmt.Sprintf(bitflyer, path, path, path, path)
	codebasePaths["exchanges bithumb"] = fmt.Sprintf(bithumb, path, path, path, path)
	codebasePaths["exchanges bitstamp"] = fmt.Sprintf(bitstamp, path, path, path, path)
	codebasePaths["exchanges bittrex"] = fmt.Sprintf(bittrex, path, path, path, path)
	codebasePaths["exchanges btcc"] = fmt.Sprintf(btcc, path, path, path, path)
	codebasePaths["exchanges btcmarkets"] = fmt.Sprintf(btcmarkets, path, path, path, path)
	codebasePaths["exchanges coinut"] = fmt.Sprintf(coinut, path, path, path, path)
	codebasePaths["exchanges exmo"] = fmt.Sprintf(exmo, path, path, path, path)
	codebasePaths["exchanges gdax"] = fmt.Sprintf(gdax, path, path, path, path)
	codebasePaths["exchanges gemini"] = fmt.Sprintf(gemini, path, path, path, path)
	codebasePaths["exchanges hitbtc"] = fmt.Sprintf(hitbtc, path, path, path, path)
	codebasePaths["exchanges huobi"] = fmt.Sprintf(huobi, path, path, path, path)
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

// returns a string for godoc package names
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

// adds all the template files
func addTemplates() error {
	glob, err := template.ParseGlob(fmt.Sprintf("readme_templates%s*", path))
	if err != nil {
		return err
	}
	_, err = glob.ParseGlob(fmt.Sprintf("sub_templates%s*", path))
	if err != nil {
		return err
	}
	_, err = glob.ParseGlob(fmt.Sprintf("exchange_readme_templates%s*", path))
	if err != nil {
		return err
	}
	_, err = glob.ParseGlob(fmt.Sprintf("general_templates%s*", path))
	if err != nil {
		return err
	}

	tmpl = glob
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

// replaces readme file
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

// creates new readme file and executes template
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
