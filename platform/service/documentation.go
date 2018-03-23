package service

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
)

const (
	gctPath = "%s%ssrc%sgithub.com%sthrasher-%sgocryptotrader%s"

	commonPath              = "%scommon%s"
	configPath              = "%sconfig%s"
	currencyPath            = "%scurrency%s"
	currencyPairPath        = "%scurrency%spair%s"
	currencySymbolPath      = "%scurrency%ssymbol%s"
	currencyTranslationPath = "%scurrency%stranslation%s"
	eventsPath              = "%sevents%s"
	exchangesPath           = "%sexchanges%s"
	exchangesNoncePath      = "%sexchanges%snonce%s"
	exchangesOrderbookPath  = "%sexchanges%sorderbook%s"
	exchangesStatsPath      = "%sexchanges%sstats%s"
	exchangesTickerPath     = "%sexchanges%sticker%s"
	platformPath            = "%splatform%s"
	portfolioPath           = "%sportfolio%s"
	smsglobalPath           = "%ssmsglobal%s"
	testdataPath            = "%stestdata%s"
	webPath                 = "%sweb%s"

	// exchange packages
	alphapoint    = "%sexchanges%salphapoint%s"
	anx           = "%sexchanges%sanx%s"
	binance       = "%sexchanges%sbinance%s"
	bitfinex      = "%sexchanges%sbitfinex%s"
	bitflyer      = "%sexchanges%sbitflyer%s"
	bithumb       = "%sexchanges%sbithumb%s"
	bitstamp      = "%sexchanges%sbitstamp%s"
	bittrex       = "%sexchanges%sbittrex%s"
	btcc          = "%sexchanges%sbtcc%s"
	btcmarkets    = "%sexchanges%sbtcmarkets%s"
	coinut        = "%sexchanges%scoinut%s"
	exmo          = "%sexchanges%sexmo%s"
	gdax          = "%sexchanges%sgdax%s"
	gemini        = "%sexchanges%sgemini%s"
	hitbtc        = "%sexchanges%shitbtc%s"
	huobi         = "%sexchanges%shuobi%s"
	itbit         = "%sexchanges%sitbit%s"
	kraken        = "%sexchanges%skraken%s"
	lakebtc       = "%sexchanges%slakebtc%s"
	localbitcoins = "%sexchanges%slocalbitcoins%s"
	okcoin        = "%sexchanges%sokcoin%s"
	okex          = "%sexchanges%sokex%s"
	poloniex      = "%sexchanges%spoloniex%s"
	wex           = "%sexchanges%swex%s"
	yobit         = "%sexchanges%syobit%s"
	liqui         = "%sexchanges%sliqui%s"

	contributorsList = "https://api.github.com/repos/thrasher-/gocryptotrader/contributors"

	licenseName     = "LICENSE"
	contributorName = "CONTRIBUTORS"
)

var (
	verbose              bool
	codebasePaths        map[string]string
	codebaseTemplatePath map[string]string
	codebaseReadme       map[string]readme
	tmpl                 *template.Template
	absolutePath         string
	pathSeparator        string
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

// StartDocumentation generates and regenerates documentation across the entire
// codebase
func StartDocumentation(Verbose bool, goPath string) {
	verbose = Verbose

	codebasePaths = make(map[string]string)
	codebaseTemplatePath = make(map[string]string)
	codebaseReadme = make(map[string]readme)

	pathSeparator = common.GetOSPathSlash()
	absolutePath = fmt.Sprintf(gctPath, goPath, pathSeparator, pathSeparator, pathSeparator, pathSeparator, pathSeparator)

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
	os.Exit(0)
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
			if verbose {
				fmt.Println("file replacement")
			}
			if err := replaceReadme(packageName); err != nil {
				return err
			}
			continue
		}
		if verbose {
			fmt.Printf("* %s Readme file NOT FOUND.\n", packageName)
		}
		if verbose {
			log.Println("file creation")
		}
		if err := createReadme(packageName); err != nil {
			return err
		}
		continue
	}
	return nil
}

// Adds paths to different potential README.md files in the codebase
func addPaths() {
	codebasePaths["common"] = fmt.Sprintf(commonPath, absolutePath, pathSeparator)
	codebasePaths["config"] = fmt.Sprintf(configPath, absolutePath, pathSeparator)
	codebasePaths["currency"] = fmt.Sprintf(currencyPath, absolutePath, pathSeparator)
	codebasePaths["currency pair"] = fmt.Sprintf(currencyPairPath, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["currency symbol"] = fmt.Sprintf(currencySymbolPath, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["currency translation"] = fmt.Sprintf(currencyTranslationPath, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["events"] = fmt.Sprintf(eventsPath, absolutePath, pathSeparator)
	codebasePaths["exchanges"] = fmt.Sprintf(exchangesPath, absolutePath, pathSeparator)
	codebasePaths["exchanges nonce"] = fmt.Sprintf(exchangesNoncePath, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges orderbook"] = fmt.Sprintf(exchangesOrderbookPath, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges stats"] = fmt.Sprintf(exchangesStatsPath, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges ticker"] = fmt.Sprintf(exchangesTickerPath, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["portfolio"] = fmt.Sprintf(portfolioPath, absolutePath, pathSeparator)
	codebasePaths["platform"] = fmt.Sprintf(platformPath, absolutePath, pathSeparator)
	codebasePaths["smsglobal"] = fmt.Sprintf(smsglobalPath, absolutePath, pathSeparator)
	codebasePaths["testdata"] = fmt.Sprintf(testdataPath, absolutePath, pathSeparator)
	codebasePaths["web"] = fmt.Sprintf(webPath, absolutePath, pathSeparator)
	codebasePaths["root"] = absolutePath

	codebasePaths["exchanges alphapoint"] = fmt.Sprintf(alphapoint, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges anx"] = fmt.Sprintf(anx, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges binance"] = fmt.Sprintf(binance, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges bitfinex"] = fmt.Sprintf(bitfinex, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges bitflyer"] = fmt.Sprintf(bitflyer, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges bithumb"] = fmt.Sprintf(bithumb, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges bitstamp"] = fmt.Sprintf(bitstamp, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges bittrex"] = fmt.Sprintf(bittrex, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges btcc"] = fmt.Sprintf(btcc, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges btcmarkets"] = fmt.Sprintf(btcmarkets, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges coinut"] = fmt.Sprintf(coinut, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges exmo"] = fmt.Sprintf(exmo, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges gdax"] = fmt.Sprintf(gdax, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges gemini"] = fmt.Sprintf(gemini, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges hitbtc"] = fmt.Sprintf(hitbtc, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges huobi"] = fmt.Sprintf(huobi, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges itbit"] = fmt.Sprintf(itbit, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges kraken"] = fmt.Sprintf(kraken, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges lakebtc"] = fmt.Sprintf(lakebtc, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges liqui"] = fmt.Sprintf(liqui, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges localbitcoins"] = fmt.Sprintf(localbitcoins, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges okcoin"] = fmt.Sprintf(okcoin, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges okex"] = fmt.Sprintf(okex, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges poloniex"] = fmt.Sprintf(poloniex, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges wex"] = fmt.Sprintf(wex, absolutePath, pathSeparator, pathSeparator)
	codebasePaths["exchanges yobit"] = fmt.Sprintf(yobit, absolutePath, pathSeparator, pathSeparator)

	codebasePaths["CONTRIBUTORS"] = absolutePath
	codebasePaths["LICENSE"] = absolutePath
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
	glob, err := template.ParseGlob(fmt.Sprintf("%splatform%sservice%sdocumentation_files%sreadme_templates%s*", absolutePath, pathSeparator, pathSeparator, pathSeparator, pathSeparator))
	if err != nil {
		return err
	}
	_, err = glob.ParseGlob(fmt.Sprintf("%splatform%sservice%sdocumentation_files%ssub_templates%s*", absolutePath, pathSeparator, pathSeparator, pathSeparator, pathSeparator))
	if err != nil {
		return err
	}
	_, err = glob.ParseGlob(fmt.Sprintf("%splatform%sservice%sdocumentation_files%sexchange_readme_templates%s*", absolutePath, pathSeparator, pathSeparator, pathSeparator, pathSeparator))
	if err != nil {
		return err
	}
	_, err = glob.ParseGlob(fmt.Sprintf("%splatform%sservice%sdocumentation_files%sgeneral_templates%s*", absolutePath, pathSeparator, pathSeparator, pathSeparator, pathSeparator))
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
